# File Watcher

一個用 Go 撰寫的檔案監控服務，可監控目錄變更並定期將檔案統計資料寫入磁碟。

## 功能

- **即時監控**：使用 `fsnotify` 監控指定目錄的檔案變更（建立、修改、刪除）
- **統計收集**：收集每個檔案的大小與最後修改時間
- **定期寫出**：每個 `batchPeriod`（預設 1 小時）將統計資料寫入 `~/.config/file_watcher/stats/YYYY-MM-DDTHH.json`
- **保留管理**：自動刪除超過 `stats_retention_days`（預設 7 天）的舊統計檔案
- **增長報告**：透過 `show` 子命令以橫條圖顯示檔案大小變化
- **Slack 通知**：當設定了環境變數時，自動將統計摘要同步發送至 Slack 頻道（多重通知器架構）

## 安裝

```bash
go build -o file_watcher .
```

## 使用方式

您可以直接使用 `go run` 指令執行，或是先將其建置為二進位檔 (binary) 後執行。

### 啟動監控服務

執行 `start` 子命令來啟動監控服務：

```bash
# 使用 go run 執行
go run main.go start

# 或是執行建置後的二進位檔
./file_watcher start
```

服務會讀取 `~/.config/file_watcher/settings.json` 的設定並開始監控。

### 查看磁碟使用量增長

執行 `show` 子命令來查看磁碟使用量增長情況：

```bash
# 使用 go run 執行
go run main.go show

# 或是執行建置後的二進位檔
./file_watcher show
```

輸出範例：

```text
磁碟使用量增長報告（初始 vs 最新）
================================================================================

/tmp/log.txt          ████████████████████████████████████████  2.5MB (+150%)
/tmp/data.json        ████████████████████████                  1.2MB (+80%)
/tmp/new.dat          █████████████████████                    (NEW)
```

### 匯出目前設定檔內容

執行 `export` 子命令來匯出目前的設定檔內容：

```bash
# 使用 go run 執行
go run main.go export

# 或是執行建置後的二進位檔
./file_watcher export
```

### 啟用 Slack 通知功能

本服務支援將定期產生的統計報告發送至 `Slack` 頻道中。您只需在啟動服務前設定以下環境變數即可：

```bash
export SLACK_BOT_TOKEN="xoxb-your-bot-token"
export SLACK_CHANNEL_ID="Cyourchannelid"
# 啟動服務
./file_watcher start
```

程式啟動時若偵測到這些環境變數，會自動加載 `SlackNotifier` 並與 `StdoutNotifier` 併用，在發送本機日誌的同時，將統計摘要同步發送至 `Slack`。

## 作為守護程序 (Daemon) 運行

本服務支援在 macOS 和 Linux 上以背景守護程序運作，並在程式被終止時自動重啟。

本專案提供了一個自動化管理腳本：

```bash
# 1. 安裝服務 (會自動偵測 OS 並生成對應設定)
./daemon.sh install

# 2. 啟動服務
./daemon.sh start

# 3. 檢查狀態與最近日誌
./daemon.sh status

# 4. 停止服務
./daemon.sh stop

# 5. 移除服務
./daemon.sh uninstall
```

更詳細的手動配置步驟（例如 `launchd` plist 設定與 Linux `systemd` user service）請參閱 [docs/daemon.md](file:///Users/shuk/projects/file_watcher/docs/daemon.md)。

## 設定檔案

設定檔案位於 `~/.config/file_watcher/settings.json`（自動建立）：

```json
{
    "watch_list": ["/tmp"],
    "exclude_list": [".git"],
    "admin": {
        "name": "admin",
        "email": "admin@localhost",
        "webhook_url": ""
    },
    "batch_period": "1h",
    "stats_retention_days": 7
}
```

### 設定欄位說明

| 欄位                   | 類型       | 說明                                              |
| ---------------------- | ---------- | ------------------------------------------------- |
| `watch_list`           | `[]string` | 要監控的目錄或檔案路徑（支援使用 `~` 代表家目錄） |
| `exclude_list`         | `[]string` | 要排除的副檔名（例如 `.git`）                     |
| `batch_period`         | `string`   | 統計寫出的間隔時間（例如 `1h`、`30m`）            |
| `stats_retention_days` | `int`      | 統計檔案的保留天數                                |

## 架構

```tree
main.go          # 進入點，初始化所有元件並處理子命令
  ├── config.go       # 讀取並驗證設定檔
  ├── export.go       # 匯出設定檔為格式化的 JSON 輸出
  ├── watcher.go      # fsnotify 包裝器，處理檔案事件
  ├── stats.go        # StatsCollector 介面與實作
  ├── scheduler.go    # 定期執行 flush 與 prune
  └── notifier.go     # 通知介面與其實作（包含 stdout 與 Slack）
```

### 核心介面（ISP + DIP）

- **StatsCollector**：`AddOrUpdate`、`Remove`、`FlushHour`、`Prune`、`Clear`
- **Notifier**：`Notify(summary string) error`（實作包含 `StdoutNotifier`、`SlackNotifier`、`MultiNotifier`）
- **WatcherOps**：`Add`、`Start`、`Close`

### 資料流向

1. `fsWatcher.Start()` 監聽 fsnotify 事件並呼叫 handler
2. Handler 呼叫 `collector.AddOrUpdate(path, size, modTime)` 記錄檔案變更
3. `Scheduler.run()` 每隔 `batchPeriod` 呼叫 `flush()`
4. `flush()` 將統計寫入 `~/.config/file_watcher/stats/YYYY-MM-DDTHH.json`
5. `Notifier.Notify()` 將統計摘要同步輸出至主控台與 `Slack`（若有配置環境變數）

### 統計資料格式

統計檔案格式（JSON）：

```json
{
    "date": "2026-05-20T01:00:00Z",
    "entries": [
        {
            "path": "/tmp/example.txt",
            "size_bytes": 1024,
            "last_modified": "2026-05-20T00:30:00Z"
        }
    ]
}
```

## 執行緒安全

`fsStatsCollector` 使用 `sync.RWMutex` 保護其內部 map，確保並發安全。

## 開發

### 建置

```bash
go build -o file_watcher .
```

### 測試

```bash
go test ./...               # 執行所有測試
go test -v ./...            # 詳細輸出
go test -race ./...         # 使用 race 檢測器執行測試
```

### 測試覆蓋的模組

- `config_test.go` — 設定檔案載入與驗證
- `stats_test.go` — 統計收集器的單元測試
- `watcher_test.go` — 檔案監控邏輯測試
- `show_test.go` — show 子命令的單元測試（formatBytes、computeGrowth）
- `notifier_test.go` — 通知器與 Mock Slack API 伺服器測試

## 預設設定

`settings.default.json` 嵌入式資源提供預設值：

```json
{
    "watch_list": ["/tmp"],
    "exclude_list": [".git"],
    "batch_period": "1h",
    "stats_retention_days": 7
}
```

## 相關檔案

- `utils/config.go` — `LoadOrCreate()` 輔助函數，用於 JSON 設定檔的讀取與自動建立
- `show.go` — `ShowCmd`、`readAllStats`、`computeGrowth`、`formatBytes`、`printBarChart`
