# File Watcher 重構計畫 — 對齊 `golang-code-quality`

## Context

本專案 (`github.com/shuk/file_watcher`) 目前所有 Go 原始檔都集中在 root 的 `package main`，僅有 `utils/` 一個 subpackage。隨著 Slack 通知、`show` 子命令、warnings 機制等功能陸續加入，`package main` 已混雜以下責任：

- 進入點與 CLI dispatch (`main.go`)
- 檔案監控 (`watcher.go`)
- 統計收集 + warnings (`stats.go`)
- 排程與 flush 流程 (`scheduler.go`)
- 通知器多型 (`notifier.go`)
- 子命令邏輯 (`show.go`、`export.go`、`test_slack.go`)
- 設定載入 (`config.go`)

這違反 `golang-code-quality` skill 中「依套件邊界劃分責任、相依方向單向往下」的原則，且累積了幾個明確的程式碼味道：fat interface (`StatsCollector` 有 8 個方法)、package-level mutable globals (`homeDirFn`、`newSlackNotifierFn`) 作為測試 stub、`Load` 與 `loadFrom` 幾乎完整重複、缺少 `context.Context` 傳遞、`fmt.Fprintf(os.Stderr, ...)` 與 `charmbracelet/log` 並存。

本計畫的目標：在不改變外部行為的前提下，將可抽象化的責任抽出為 root 下的 subpackage，套用 ISP/DIP，並消除上述程式碼味道。

## 範圍 (Scope)

包含：

- 套件分層（root + subpackages，不動到 `internal/` 結構）
- 介面切分（拆 fat interface）
- DI 清理（移除測試用 globals）
- `context.Context` 傳遞
- 錯誤處理 / 日誌一致化
- 程式碼去重與小修

不包含 (Out of Scope，留作後續 PR)：

- Security 修補：`main.go:101` 的 token 印到 stdout、`utils/config.go:47` 把 config 內容寫進 log — 本次 refactor 一律保留原樣，由獨立 PR 處理
- 任何行為變更或新功能
- 將 root 改為 `internal/` 或 `cmd/` 結構

## 目標套件佈局

維持 root 為 `package main`，將「可抽象化、有獨立責任邊界」的程式碼抽出為 subpackage：

```
.
├── main.go              # 進入點 + CLI dispatch (薄)
├── bootstrap.go         # 集中所有具體型別 wiring (DI 唯一入口) — 新檔
├── config/              # 設定載入 + Settings 定義
│   └── config.go
├── watcher/             # fsnotify 包裝器
│   └── watcher.go
├── stats/               # 統計收集 (純資料責任)
│   ├── collector.go
│   └── entry.go         # StatEntry / StatFile / StatFileEntry 合併
├── warning/             # 從 stats 抽出的 warning bag (SRP)
│   └── sink.go
├── scheduler/           # 排程 + flush 流程
│   └── scheduler.go
├── notify/              # 通知介面 + 各 Notifier 實作
│   ├── notifier.go      # Notifier 介面 + Multi
│   ├── stdout.go
│   └── slack.go
├── show/                # show 子命令（讀檔、計算成長、印 bar chart）
│   ├── show.go
│   └── growth.go
└── utils/               # 既有 LoadOrCreate；保持窄定義
    └── config.go
```

`export.go`、`test_slack.go` 留在 root，因為它們只是 CLI 子命令薄殼，呼叫各 subpackage。Module path 不變，所有 import 走 `github.com/shuk/file_watcher/<pkg>`。

命名取捨：subpackage 命名遵循 skill 規則（小寫、單數、無底線、避免 stutter）。`scheduler.New()` 回傳未 export 的具體型別以避開 `scheduler.Scheduler` 的 stutter；`notify.Notifier`、`stats.Collector`、`watcher.Watcher` 都不 stutter（套件名為動詞 / 領域名，型別名描述角色）。

相依方向（單向往下）：

```
main → bootstrap → {scheduler, watcher, notify, show, config, stats, warning}
scheduler → {stats, warning, notify}
watcher → (僅 fsnotify)
notify → (僅 slack-go)
show → stats
config → utils
```

絕對禁止：任一 subpackage import root、或 sibling import sibling 形成循環。

## 主要重構動作

### 1. 套件抽出

對每個 subpackage：

- 把對應檔案搬入新目錄，改 `package` 宣告
- 該檔案內原本 unexported 但會被跨檔呼叫的識別字（例如 `fsStatsCollector.AddOrUpdate`）需重新評估 export
- 在 root 對應檔案保留 import + 薄包裝（如有必要，否則整檔刪除）
- 更新所有 test 檔的 package 宣告與 import

最具代表性的搬移：

- `scheduler.go` → `scheduler/scheduler.go`，型別 `Scheduler` 改名為未 export 的 `scheduler` 並透過 `New()` 建構，避免 `scheduler.Scheduler` stutter
- `notifier.go` 拆成 `notify/notifier.go` (介面 + Multi) + `notify/stdout.go` + `notify/slack.go`
- `stats.go` 拆成 `stats/collector.go` (collector) + `stats/entry.go` (純資料型別)，warning 相關方法搬入 `warning/sink.go`

### 2. 介面切分（ISP）

`stats.StatsCollector` 目前有 8 個方法，違反 ISP。按消費端切分：

- `stats.Recorder` — `AddOrUpdate(path string, size int64, modTime time.Time)`、`Remove(path string)`（watcher handler 消費）
- `stats.Flusher` — `FlushHour(ctx) error`、`Clear()`、`Prune(ctx, retentionDays int) error`（scheduler 消費）

warning 完全脫離 `stats`，移至 `warning.Sink` 介面：`Add(msg string)`、`Drain() []string`（drain 取代既有 GetWarnings + ClearWarnings 兩呼叫的競態風險）。

關鍵點：介面定義在 `消費端` 套件，不在 `生產端`。例如 `scheduler/` 內部宣告自己需要的 `flusher` 與 `warningDrainer` 介面，而非 `stats/` 主動 export。

### 3. DI 清理（DIP）

移除兩個 package-level mutable globals：

- `config.go:17` 的 `var homeDirFn = func() string { return os.Getenv("HOME") }` — 改為 `config.Loader` 結構，建構時注入 `homeDir string`，測試以 `config.NewLoader(t.TempDir())` 替換
- `test_slack.go:10` 的 `var newSlackNotifierFn` — 改為 `test_slack.go` 接受一個 notifier factory 參數，由 `bootstrap.go` 注入

新增 `bootstrap.go`（root）作為唯一具體型別交會點：

```go
// bootstrap.go (示意)
package main

func wire(cfg *config.Settings) (*runtime, error) {
    homeDir, _ := os.UserHomeDir()
    statsDir := filepath.Join(homeDir, ".config", "file_watcher", "stats")

    w, err := watcher.New(cfg.ExcludeList)
    // ...
    collector := stats.NewCollector(statsDir)
    warnings := warning.NewSink()
    notifier := buildNotifier(cfg) // 內部組 Stdout + Slack + Multi
    sched := scheduler.New(collector, warnings, notifier, period, cfg.StatsRetentionDays)
    return &runtime{watcher: w, scheduler: sched, ...}, nil
}
```

`main.go` 縮為純 CLI dispatch + `wire()` + signal handling。

### 4. Context 傳遞

目前完全沒有 `context.Context` 流經。本次補上：

- `notify.Notifier.Notify(ctx context.Context, summary string) error` — Slack HTTP 呼叫改用 `PostMessageContext`
- `watcher.Watcher.Start(ctx context.Context, handler watcher.Handler) error` — 移除目前 `Start()` 內部 `context.WithCancel(context.Background())` 的反模式，改為接受外部 ctx
- `scheduler.New(...).Start(ctx context.Context) error` — `ticker` 迴圈用 `ctx.Done()` 取代自製 `stop chan`
- `stats.Flusher.FlushHour(ctx) error`、`Prune(ctx, days) error`

`main.go` 建立一個 ctx，於 `signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)` 接 cancel；scheduler/watcher 收到 cancel 後優雅停機，取代既有 `FlushNow()` + `close(done)` 的雙軌機制。

### 5. 錯誤處理一致化

- 全面以 `charmbracelet/log` 取代散佈在 `scheduler.go:57`、`watcher.go:179`、`main.go` 的 `fmt.Fprintf(os.Stderr, ...)`。`main.go` 的 fatal exit 改用 `log.Fatal()`
- `notify/notifier.go` 的 `MultiNotifier.Notify` 目前用 `fmt.Errorf("some notifiers failed: %v", errs)` 失去 error chain — 改用 `errors.Join(errs...)`（Go 1.20+，本專案 `go 1.25.0`）
- 所有錯誤回傳一律 `%w` wrap，禁用 `%v`
- `show.go:194` 的 `parseTime` 吞掉 error — 改為回傳 `(time.Time, error)` 或直接刪除（檢查呼叫處）

### 6. 程式碼去重與小修

- `config.go` 的 `Load()` 與 `loadFrom()` 幾乎完整重複（defaults / parse / validate 邏輯各重複一次）— 合併為一個內部 helper `parse(data []byte) (*Settings, error)`，`Load()` 與 `loadFrom()` 各自只負責取得 bytes
- `stats.go` 的 `StatEntry` (Size + LastModified) 與 `StatFileEntry` (Path + Size + LastModified) 是同一個 struct 的兩種視角 — 統一為 `stats.Entry`，map value 與檔案 entries 共用
- `utils/config.go:57` 自寫的 `dir()` 函數重新實作 `filepath.Dir` — 改用標準庫
- `stats.go:163` 的 `defaultStatsDir()` 直接呼叫 `os.Getenv("HOME")` — 改為由 `bootstrap.go` 計算好後注入 collector
- `config.go` 的 `Admin` struct 保留但加上 doc comment：`// Admin is reserved for future admin-notification integration; currently unread.`，避免被誤刪

### 7. 明確不動的部分

- `main.go:101` 的 `fmt.Println("Slack 憑證已偵測 ...", "token", slackToken, ...)` — 保留原樣（安全修補留後續 PR）
- `utils/config.go:47` 的 `log.Info("Loading config from ", path, string(data))` — 保留原樣（同上）
- `settings.default.json`、CLI 子命令名稱、config 檔路徑、stats 檔命名規則 — 對外行為一律不動
- `scripts/manage-daemon.sh`、`docs/daemon.md` — 不在 refactor 範圍

## 關鍵檔案清單

需要新建：

- `bootstrap.go` (root)
- `config/config.go`、`watcher/watcher.go`、`stats/collector.go`、`stats/entry.go`、`warning/sink.go`、`scheduler/scheduler.go`、`notify/notifier.go`、`notify/stdout.go`、`notify/slack.go`、`show/show.go`、`show/growth.go`
- 以及對應 `*_test.go`

需要大幅修改：

- `main.go` — 縮為 CLI dispatch + wire + signal handling
- `config.go` (root) — 刪除或縮為對 `config/` 的 re-export（傾向直接刪除，由 import 取代）

需要小修：

- `export.go`、`test_slack.go` — 改為依賴新套件並接受注入的 factory
- 所有 `*_test.go` — package 宣告與 import 調整

可繼續復用：

- `utils/LoadOrCreate` — 保留作為 config 套件的依賴
- `fsnotify`、`slack-go/slack`、`charmbracelet/log` — 依賴關係不變

## 介面契約範例（供執行階段參考）

```go
// notify/notifier.go
package notify

type Notifier interface {
    Notify(ctx context.Context, summary string) error
}

type Multi struct{ ns []Notifier }
func NewMulti(ns ...Notifier) *Multi { return &Multi{ns} }
func (m *Multi) Notify(ctx context.Context, s string) error {
    var errs []error
    for _, n := range m.ns {
        if err := n.Notify(ctx, s); err != nil {
            errs = append(errs, err)
        }
    }
    return errors.Join(errs...)
}
```

```go
// stats/collector.go
package stats

type Recorder interface {
    AddOrUpdate(path string, size int64, modTime time.Time)
    Remove(path string)
}

type Flusher interface {
    FlushHour(ctx context.Context) error
    Clear()
    Prune(ctx context.Context, retentionDays int) error
}

// Collector 同時實作 Recorder 與 Flusher；不 export 為單一 fat interface。
type Collector struct { /* ... */ }
```

## 驗證步驟

1. `go build -o file_watcher .` — 確認可編譯
2. `go vet ./...` — 確認無靜態警告
3. `go test ./...` — 全部既有測試通過
4. `go test -race ./...` — race detector 通過（特別注意 warning sink 與 collector 的並發測試）
5. 手動驗證行為不變：
   - `./file_watcher show`（空 stats 目錄與有 stats 目錄各跑一次）
   - `./file_watcher export`（輸出與 refactor 前 byte-for-byte 一致）
   - `./file_watcher test-slack`（若有 token 環境變數）
   - `./file_watcher`（背景跑 1 分鐘，調短 `batch_period` 為 `10s`，確認有 flush + notify）
6. `git diff --stat main..HEAD` 抽樣檢視 — 確認沒有意外動到 settings.default.json、daemon 腳本、docs
7. 重新讀過 `main.go` 與 `bootstrap.go` — 確認 `main.go` 不再持有任何業務邏輯，所有 wiring 都在 `bootstrap.go`

## 預期效益

- `package main` 從 ~10 個檔案、混雜 6 種責任縮為 ~3 個檔案（main、bootstrap、CLI 薄殼），符合 SRP
- `StatsCollector` 8 方法 fat interface 拆成 2 個 ≤3 方法的 focused interface，消費端只看得到自己需要的方法（ISP）
- 兩個測試用 global stub 消失，所有相依透過 constructor 注入（DIP）
- `context.Context` 從入口貫穿至所有 I/O 路徑，支援優雅取消
- 後續若要新增 `EmailNotifier`、`WebhookNotifier`、或新的子命令，只需新增檔案而不動既有程式碼（OCP）
