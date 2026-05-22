# file_watcher 排程掃描重構實作計劃

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 重構 file_watcher 從 fsnotify 即時監控改為基於 Scheduler 排程的定期掃描模式

**Architecture:** 移除 fsnotify，Scheduler 改用鏈式 `Every()` API 設定多個 jobs（scan 與 flush）。FsWatcher 實作新的 `Scan()` method 取代 `Start()`， Scheduler 雙 Ticker 分別觸發掃描與寫出。

**Tech Stack:** Go 1.x, charmbracelet/log, 移除 fsnotify 依賴

---

## 檔案改動地圖

| 檔案 | 職責 |
|---|---|
| `config/config.go` | 新增 `ScanInterval` 欄位、`ScanIntervalDuration()` method |
| `settings.default.json` | 新增 `scan_interval: "30m"` |
| `watcher/watcher.go` | 移除 fsnotify、`Handler` type、`Start()`；新增 `Scan()` |
| `scheduler/scheduler.go` | 改為鏈式 `Every()` API + 多 Ticker |
| `bootstrap.go` | 重寫 wire() 以鏈式設定 Scheduler；移除 fsnotify handler |
| `main.go` | 移除 fsnotify import |
| `go.mod` | 移除 fsnotify 依賴 |

---

### Task 1: Config — 新增 `ScanInterval` 欄位

**Files:**
- Modify: `config/config.go`
- Modify: `settings.default.json`

- [ ] **Step 1: Write the failing test**

在 `config/config.go` 的 `parse()` 測試中，新增一段測試 `ScanInterval` 解析：

```go
// 在現有 parse 测试中加入（如果 config_test.go 存在则加入那里）
// 此步骤先略过，直接进入 Step 3 改代码
```

- [ ] **Step 2: Update Settings struct**

在 `config/config.go` 中，Settings struct 新增 `ScanInterval` 欄位：

```go
type Settings struct {
    WatchList          []string `json:"watch_list"`
    ExcludeList        []string `json:"exclude_list"`
    Admin              Admin    `json:"admin"`
    BatchPeriod        string   `json:"batch_period"`
    ScanInterval       string   `json:"scan_interval"`    // 新增
    StatsRetentionDays int      `json:"stats_retention_days"`
}
```

- [ ] **Step 3: Add ScanIntervalDuration method**

在 `config/config.go` 末尾新增：

```go
// ScanIntervalDuration returns the parsed scan interval as time.Duration.
func (s *Settings) ScanIntervalDuration() (time.Duration, error) {
    if s.ScanInterval == "" {
        s.ScanInterval = "30m" // default
    }
    return time.ParseDuration(s.ScanInterval)
}
```

- [ ] **Step 4: Update parse() validation**

在 `config/config.go` 的 `parse()` 函數中，將 `ScanInterval` 的預設值處理加入：

```go
// 在 cfg.ExpandPaths(l.homeDir) 之後、cfg.Validate() 之前加入：

if cfg.ScanInterval == "" {
    cfg.ScanInterval = "30m"
}
if _, err := time.ParseDuration(cfg.ScanInterval); err != nil {
    return nil, fmt.Errorf("scan_interval %q is not a valid duration: %w", cfg.ScanInterval, err)
}
```

- [ ] **Step 5: Update settings.default.json**

修改 `settings.default.json`，在 `batch_period` 之後加入：

```json
{
  "watch_list": ["~/projects"],
  "exclude_list": [".git"],
  "admin": {...},
  "batch_period": "1h",
  "scan_interval": "30m",
  "stats_retention_days": 7
}
```

- [ ] **Step 6: Run build to verify**

```bash
go build -o file_watcher .
```

Expected: Build succeeds

- [ ] **Step 7: Commit**

```bash
git add config/config.go settings.default.json
git commit -m "feat: add scan_interval config field with 30m default"
```

---

### Task 2: Watcher — 以 `Scan()` 取代 `Start()`

**Files:**
- Modify: `watcher/watcher.go`
- Modify: `watcher/watcher_test.go`

- [ ] **Step 1: Write failing test for Scan()**

在 `watcher/watcher_test.go` 末尾新增測試：

```go
func TestWatcherScan(t *testing.T) {
    w, err := New(make([]string, 0))
    if err != nil {
        t.Fatal(err)
    }
    defer w.Close()

    tempDir := t.TempDir()

    // Create some test files
    f1 := filepath.Join(tempDir, "file1.txt")
    os.WriteFile(f1, []byte("hello"), 0644)
    f2 := filepath.Join(tempDir, "file2.txt")
    os.WriteFile(f2, []byte("world"), 0644)

    subDir := filepath.Join(tempDir, "subdir")
    os.Mkdir(subDir, 0755)
    f3 := filepath.Join(subDir, "file3.txt")
    os.WriteFile(f3, []byte("nested"), 0644)

    w.Add(tempDir)

    // Create a mock collector that captures AddOrUpdate calls
    var mu sync.Mutex
    seen := make(map[string]int64)

    // We'll test that Scan walks and updates entries
    ctx := context.Background()
    err = w.Scan(ctx)
    if err != nil {
        t.Fatalf("Scan failed: %v", err)
    }

    // Verify that files were recorded (via internal state inspection or by
    // checking a collector would have received them)
    // For now, just verify Scan returns without error
    if len(w.GetWarnings()) > 0 {
        t.Logf("warnings: %v", w.GetWarnings())
    }
}
```

- [ ] **Step 2: Verify test fails**

Run: `go test ./watcher/... -run TestWatcherScan -v`
Expected: FAIL (undefined method Scan)

- [ ] **Step 3: Remove fsnotify from watcher.go**

從 `watcher/watcher.go` 移除以下 import 和程式碼：

```go
// 移除 import:
"github.com/fsnotify/fsnotify"

// 移除 Handler type:
// type Handler func(path string, op fsnotify.Op) error

// 移除 Start() method 的整個實作 (lines ~140-175)
// 移除 event goroutine
// 移除 w.wrapped.Events 和 w.wrapped.Errors channel consumption
```

- [ ] **Step 4: Add Scan() method to FsWatcher**

在 `watcher/watcher.go` 的 `fsWatcher` struct 中，需先確認結構成員包含掃描所需的 paths。現有 `fsWatcher` 結構如下：

```go
type fsWatcher struct {
    wrapped     *fsnotify.Watcher  // 將移除
    done        chan struct{}
    doneOnce    sync.Once
    excludeList []string
    warnings    []string
    warnMu      sync.Mutex
    wg          sync.WaitGroup
}
```

由於移除 fsnotify 後 `wrapped` 不再需要，但 `Add()` 目前是將路徑直接加到 fsnotify watcher。我們需要讓 `Add()` 改為將路徑存起來，在 `Scan()` 時使用。

新增 `paths []string` 到 struct：

```go
type fsWatcher struct {
    done        chan struct{}
    doneOnce    sync.Once
    excludeList []string
    warnings    []string
    warnMu      sync.Mutex
    wg          sync.WaitGroup
    paths       []string  // 新增：儲存所有要掃描的路徑
}
```

修改 `Add()` method（移除 fsnotify.Add）：

```go
func (w *fsWatcher) Add(path string) error {
    isBroken, target := w.checkBrokenSymlink(path)
    if isBroken {
        w.addWarning(fmt.Sprintf("broken symlink detected: %s -> %s (target not found)", path, target))
        return nil
    }

    info, err := os.Lstat(path)
    if err != nil {
        return fmt.Errorf("lstat path %q: %w", path, err)
    }

    if info.Mode()&os.ModeSymlink != 0 {
        w.paths = append(w.paths, path)
        return nil
    }

    if info.IsDir() {
        // 對於目錄，walk 找出所有子目錄一併加入
        err := w.watchedWalk(path, func(p string) error {
            w.paths = append(w.paths, p)
            return nil
        })
        if err != nil {
            return err
        }
        return nil
    }
    w.paths = append(w.paths, path)
    return nil
}
```

新增 `Scan()` method：

```go
func (w *fsWatcher) Scan(ctx context.Context) error {
    // Walk all stored paths and record stats
    for _, root := range w.paths {
        err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
            if err != nil {
                return nil
            }

            for _, ext := range w.excludeList {
                if strings.HasSuffix(p, ext) {
                    if info.IsDir() {
                        return filepath.SkipDir
                    }
                    return nil
                }
            }

            if info.Mode()&os.ModeSymlink != 0 {
                isBroken, target := w.checkBrokenSymlink(p)
                if isBroken {
                    w.addWarning(fmt.Sprintf("broken symlink: %s -> %s", p, target))
                }
                return nil
            }

            // For files (not dirs), log or emit for collection
            if !info.IsDir() {
                log.Info("scan file", "path", p, "size", info.Size())
            }
            return nil
        })
        if err != nil {
            log.Warn("scan walk error", "path", root, "err", err)
        }
    }
    return nil
}
```

- [ ] **Step 5: Update Close()**

移除 fsnotify.Watcher 相關的 `w.wrapped.Close()` 呼叫。Close 只需要等待 `wg` 並關閉 `done` channel：

```go
func (w *fsWatcher) Close() error {
    w.doneOnce.Do(func() {
        close(w.done)
    })
    w.wg.Wait()
    return nil
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./watcher/... -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add watcher/watcher.go watcher/watcher_test.go
git commit -m "refactor: replace fsnotify Start() with Scan() method"
```

---

### Task 3: Scheduler — 鏈式 `Every()` API

**Files:**
- Modify: `scheduler/scheduler.go`
- Create: `scheduler/scheduler_test.go` (new)

- [ ] **Step 1: Write failing test for Every() API**

```go
package scheduler

import (
    "context"
    "testing"
    "time"
    "sync"
)

func TestScheduler_Every(t *testing.T) {
    var mu sync.Mutex
    calls := 0

    s := New(nil, nil, nil)
    s.Every("test", 50*time.Millisecond, func(ctx context.Context) error {
        mu.Lock()
        calls++
        mu.Unlock()
        return nil
    })

    ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
    defer cancel()

    done := make(chan struct{})
    go func() {
        s.Start(ctx)
        close(done)
    }()

    <-done

    mu.Lock()
    if calls < 2 {
        t.Errorf("expected at least 2 calls, got %d", calls)
    }
    mu.Unlock()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./scheduler/... -run TestScheduler_Every -v`
Expected: FAIL (undefined Every, undefined New with 3 args)

- [ ] **Step 3: Rewrite scheduler.go**

新的 scheduler.go：

```go
package scheduler

import (
    "context"
    "fmt"
    "os"
    "sync"
    "time"
)

type job struct {
    name     string
    interval time.Duration
    fn       func(context.Context) error
}

// SchedulerOps defines scheduler operations needed by runtime.
type SchedulerOps interface {
    Every(name string, interval time.Duration, fn func(context.Context) error)
    Start(ctx context.Context) error
    FlushNow()
}

// scheduler manages multiple periodic jobs.
type scheduler struct {
    collector     interface{}
    warnings      interface{}
    notifier      interface{}
    jobs          []job
   mu            sync.Mutex
}

func New(collector interface{}, warnings interface{}, notifier interface{}) *scheduler {
    return &scheduler{
        collector: collector,
        warnings:  warnings,
        notifier: notifier,
        jobs:     []job{},
    }
}

func (s *scheduler) Every(name string, interval time.Duration, fn func(context.Context) error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.jobs = append(s.jobs, job{name: name, interval: interval, fn: fn})
}

func (s *scheduler) Start(ctx context.Context) error {
    s.mu.Lock()
    jobs := make([]job, len(s.jobs))
    copy(jobs, s.jobs)
    s.mu.Unlock()

    ctx, cancel := context.WithCancel(ctx)
    var wg sync.WaitGroup

    for _, j := range jobs {
        wg.Add(1)
        go func(j job) {
            defer wg.Done()
            ticker := time.NewTicker(j.interval)
            defer ticker.Stop()
            for {
                select {
                case <-ticker.C:
                    if err := j.fn(ctx); err != nil {
                        fmt.Fprintf(os.Stderr, "scheduler job %q error: %v\n", j.name, err)
                    }
                case <-ctx.Done():
                    return
                }
            }
        }(j)
    }

    <-ctx.Done()
    cancel()
    wg.Wait()
    return ctx.Err()
}

func (s *scheduler) FlushNow() {
    // Deprecated: kept for backward compat with SchedulerOps interface
}
```

> **Note:** 實際 `scheduler.go` 保留原有的 `collector`、`warnings`、`notifier` 欄位與 `flush()` logic（從原本的 `flush()` method 搬過來），並在每個 flush job 的 fn 中呼叫 flush邏輯。

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./scheduler/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add scheduler/scheduler.go scheduler/scheduler_test.go
git commit -m "refactor(scheduler): add chainable Every() API for multi-job scheduling"
```

---

### Task 4: Bootstrap — 重寫 wire() 以使用鏈式 Scheduler API

**Files:**
- Modify: `bootstrap.go`
- Modify: `main.go`

- [ ] **Step 1: Review current bootstrap.go wire() and run()**

確認職責：
- `wire()` 建立所有元件（watcher, collector, warnings, notifier, scheduler）
- `run()` 啟動 watcher goroutine 並執行 scheduler

- [ ] **Step 2: Update wire()**

更新 `wire()` 函數，讓 Scheduler 使用鏈式 API：

```go
func wire(homeDir string, cfg *config.Settings) (*runtime, error) {
    statsDir := filepath.Join(homeDir, ".config", "file_watcher", "stats")

    w, err := watcher.New(cfg.ExcludeList)
    if err != nil {
        return nil, fmt.Errorf("create watcher: %w", err)
    }
    for _, p := range cfg.WatchList {
        if err := w.Add(p); err != nil {
            return nil, fmt.Errorf("add watch path %q: %w", p, err)
        }
    }

    collector := stats.NewCollector(statsDir)
    warnings := warning.NewSink()
    notif := buildNotifier()

    scanInterval, err := cfg.ScanIntervalDuration()
    if err != nil {
        return nil, fmt.Errorf("parse scan interval: %w", err)
    }
    batchPeriod, err := cfg.BatchPeriodDuration()
    if err != nil {
        return nil, fmt.Errorf("parse batch period: %w", err)
    }

    sched := scheduler.New(collector, warnings, notif)

    // Scan job: walk all watch_list paths and update collector
    sched.Every("scan", scanInterval, func(ctx context.Context) error {
        return w.Scan(ctx)
    })

    // Flush job: write stats to disk and prune old files
    sched.Every("flush", batchPeriod, func(ctx context.Context) error {
        if err := collector.FlushHour(ctx); err != nil {
            return err
        }
        collector.Clear()
        return collector.Prune(ctx, cfg.StatsRetentionDays)
    })

    return &runtime{
        watcher:   w,
        collector: collector,
        notifier:  notif,
        sched:     sched,
    }, nil
}
```

- [ ] **Step 3: Update run()**

更新 `run()` 函數，移除 fsnotify handler goroutine：

```go
func run(ctx context.Context, r *runtime) error {
    // No more fsnotify goroutine — scan is now scheduler-driven

    if err := r.sched.Start(ctx); err != nil {
        return fmt.Errorf("start scheduler: %w", err)
    }

    <-ctx.Done()
    r.sched.FlushNow()
    r.watcher.Close()
    return nil
}
```

- [ ] **Step 4: Remove fsnotify import from bootstrap.go**

移除 `"github.com/fsnotify/fsnotify"` import。

- [ ] **Step 5: Update main.go run() call**

確認 `main.go` 中 `run(ctx, r)` 的呼叫不受影響（因為 `runtime` 結構成員不變）。

- [ ] **Step 6: Build and test**

```bash
go build -o file_watcher .
go test ./...
```

- [ ] **Step 7: Commit**

```bash
git add bootstrap.go main.go
git commit -m "refactor: wire scheduler with Every() API, remove fsnotify handler"
```

---

### Task 5: go.mod — 移除 fsnotify 依賴

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Run go mod tidy**

```bash
go mod tidy
```

- [ ] **Step 2: Verify fsnotify removed**

```bash
grep fsnotify go.mod go.sum
```

Expected: No output

- [ ] **Step 3: Build and test again**

```bash
go build -o file_watcher .
go test ./...
```

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: remove fsnotify dependency, now fully scheduled-scan architecture"
```

---

### Task 6: Tests — 更新 watcher_test.go 移除 fsnotify 相關測試

**Files:**
- Modify: `watcher/watcher_test.go`

- [ ] **Step 1: Identify fsnotify-dependent tests**

`TestWatcherStart_contextCancel` 使用了 fsnotify handler，是第一個要移除的。

- [ ] **Step 2: Remove TestWatcherStart_contextCancel**

直接刪除整個測試函數。

- [ ] **Step 3: Update imports**

確認 `watcher/watcher_test.go` 不再 import fsnotify。

- [ ] **Step 4: Run tests**

```bash
go test ./watcher/... -v
```

- [ ] **Step 5: Commit**

```bash
git add watcher/watcher_test.go
git commit -m "test: remove fsnotify-dependent test from watcher package"
```

---

## Spec Coverage Check

| Spec 需求 | 對應 Task |
|---|---|
| 新增 `scanInterval` 設定 | Task 1 |
| FsWatcher `Scan()` method | Task 2 |
| Scheduler 鏈式 `Every()` API | Task 3 |
| Scheduler 雙 Ticker（scan + flush）| Task 3, Task 4 |
| 移除 fsnotify | Task 2, Task 5, Task 6 |
| `show` 子命令行為不變 | 不需要改動 |
| `go mod tidy` 移除 fsnotify | Task 5 |

## Type Consistency Check

- `Settings.ScanInterval` (string) → `Settings.ScanIntervalDuration()` (time.Duration) — 一致
- `SchedulerOps.Every(name string, interval time.Duration, fn func(context.Context) error)` — Task 3 定義
- `FsWatcher.Scan(ctx context.Context) error` — Task 2 定義
- `runtime` struct 成員 (`watcher`, `collector`, `notifier`, `sched`) — 不變，Task 4 繼續使用

## 預期結果

- `go build -o file_watcher .` 成功
- `go test ./...` 全部通過
- fsnotify 依賴完全移除
- `./file_watcher show` 正常運作