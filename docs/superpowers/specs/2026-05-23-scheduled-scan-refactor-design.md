# file_watcher 排程掃描重構設計

**Date:** 2026-05-23
**Status:** Draft

## 1. 目標

將 file_watcher 從 fsnotify 即時監控模式重構為**基於排程的主動掃描模式**。

- **舊行為**：fsnotify 監聽檔案系統即時事件，每次事件都呼叫 `collector.AddOrUpdate`
- **新行為**：Scheduler 每隔 `scanInterval` 自動完整掃描 `watch_list` 中所有目錄

## 2. 設計決策

| 決策 | 選擇 |
|---|---|
| `scanInterval` 預設值 | `30m` |
| `scanInterval` vs `batchPeriod` 關係 | 兩者完全獨立：`scanInterval` 控制掃描頻率，`batchPeriod` 控制統計寫出頻率 |
| fsnotify 處理 | 完全移除，不保留即時監控模式 |
| Scheduler 擴展方式 | 鏈式 API (`Every(name, interval, fn)`) |

## 3. 新介面定義

### 3.1 Watcher 介面（改寫）

```go
type Watcher interface {
    Add(path string) error
    Scan(ctx context.Context) error
    Close() error
    GetWarnings() []string
}
```

**移除**：`Start(ctx context.Context, handler Handler) error`、`Handler` type

**新增**：`Scan(ctx context.Context) error` — 對所有已 `Add()` 的路徑執行 `filepath.Walk`，對每個檔案呼叫 Collector

### 3.2 Scheduler 鏈式 API

```go
type SchedulerOps interface {
    Every(name string, interval time.Duration, fn func(context.Context) error)
    Start(ctx context.Context) error
    FlushNow()
}
```

使用範例：

```go
sched.Every("scan", scanInterval, func(ctx context.Context) error {
    return w.Scan(ctx)
})
sched.Every("flush", batchPeriod, func(ctx context.Context) error {
    if err := collector.FlushHour(ctx); err != nil {
        return err
    }
    collector.Clear()
    return collector.Prune(ctx, retentionDays)
})
```

**Why:** 鏈式 API 讓新增 job 只需要一行，不改動 Scheduler 結構本身。

## 4. 設定檔改動

### 4.1 settings.default.json（嵌入式）

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

### 4.2 Settings struct

```go
type Settings struct {
    WatchList          []string `json:"watch_list"`
    ExcludeList        []string `json:"exclude_list"`
    Admin              Admin    `json:"admin"`
    BatchPeriod        string   `json:"batch_period"`
    ScanInterval       string   `json:"scan_interval"`
    StatsRetentionDays int      `json:"stats_retention_days"`
}
```

Validation：
- 若 `ScanInterval` 為空，預設 `30m`
- 若 `BatchPeriod` 為空，預設 `1h`

## 5. 資料流向

```
Scheduler.run()
  ├── scan ticker (scanInterval) ──→ FsWatcher.Scan(ctx)
  │                                    └── filepath.Walk(watch_list)
  │                                        └── collector.AddOrUpdate(path, size, modTime)
  │
  └── flush ticker (batchPeriod) ──→ collector.FlushHour(ctx)
                                        └── collector.Clear()
                                        └── collector.Prune(ctx, retentionDays)
                                            └── notifier.Notify(summary)
```

## 6. 實作改動清單

| 檔案 | 改動 |
|---|---|
| `config/config.go` | 新增 `ScanInterval` 欄位、`ScanIntervalDuration()` method |
| `settings.default.json` | 新增 `scan_interval: "30m"` |
| `watcher/watcher.go` | 移除 fsnotify、`Handler` type、`Start()`；新增 `Scan()` method |
| `scheduler/scheduler.go` | 改為雙 Ticker + 鏈式 `Every()` API |
| `bootstrap.go` | 重寫 `wire()`；移除 fsnotify handler goroutine |
| `main.go` | `run()` 移除 goroutine 啟動 watcher |
| `go.mod` | 移除 `github.com/fsnotify/fsnotify` dependency |

## 7. Scanner vs Watcher 分離（長期待辦）

未來若需要多個 Scanner（例如不同目錄用不同掃描策略），可將 Scanner 職責從 Watcher 介面分離：

```go
type Scanner interface {
    Scan(ctx context.Context) error
}

type WatcherOps interface {
    Add(path string) error
    Close() error
    GetWarnings() []string
}
```

此為長期待辦，本文不作實作。

## 8. 測試策略

- `watcher_test.go`：測試新的 `Scan()` method，驗證 `exclude_list` 正確過濾
- `scheduler_test.go`：測試鏈式 API 與雙 Ticker 邏輯（需Mock時間）
- `bootstrap_test.go`：測試 wire 函數正確建立所有元件

## 9. 預期產出

- 新的 `FsWatcher.Scan()` 完整掃描 `watch_list` 並更新 Collector
- Scheduler 支援雙（以上）ticker，透過 `Every()` 鏈式設定
- 完全移除 fsnotify 依賴
- `show` 子命令行為不變