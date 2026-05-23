# File Watcher

一個用 Go 撰寫的檔案監控服務，可監控目錄變更並定期將檔案統計資料寫入磁碟。

## 功能

- `即時監控`：使用 `fsnotify` 監控指定目錄的檔案變更（建立、修改、刪除）
- `統計收集`：收集每個檔案的大小與最後修改時間
- `定期寫出`：每個 `batchPeriod`（預設 1 小時）將統計資料寫入 `~/.config/file_watcher/stats/YYYY-MM-DDTHH.json`
- `保留管理`：自動刪除超過 `stats_retention_days`（預設 7 天）的舊統計檔案
- `增長報告`：透過 `show` 子命令以橫條圖顯示檔案大小變化，直接執行 `./file_watcher` 亦會預設輸出報告
- `Slack 通知`：當設定了環境變數時，自動將統計摘要同步發送至 Slack 頻道（多重通知器架構）

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

執行 `show` 子命令來查看磁碟使用量增長情況（或直接執行 `./file_watcher`）：

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

| 欄位 | 類型 | 說明 |
| --- | --- | --- |
| `watch_list` | `[]string` | 要監控的目錄或檔案路徑（支援使用 `~` 代表家目錄） |
| `exclude_list` | `[]string` | 要排除的副檔名或目錄名 |
| `batch_period` | `string` | 統計寫出的間隔時間（例如 `1h`、`30m`） |
| `stats_retention_days` | `int` | 統計檔案的保留天數 |

## 架構

```tree
main.go          # 進入點，調用 cmd.Execute()
  ├── cmd/            # 命令列子命令 (RootCmd, StartCmd, ExportCmd, ShowCmd)
  ├── config/         # 讀取並驗證設定檔 (config.go)
  ├── handler/        # 生命週期管理器 (Wire, Run) 驅動排程與事件 (runner.go)
  └── svc/            # 整合服務層，包含 watcher, collector, sink, show 邏輯
```

### 核心介面與定義（ISP + DIP）

- `svc.Collector`：實現 `Recorder` 與 `Flusher` 介面（管理 `Entry` 寫入與歷史統計修剪）
- `notify.Notifier`：通知介面（實作包含 `StdoutNotifier`、`SlackNotifier`、`Multi`）
- `svc.Watcher`：監控檔案目錄介面與 `fsWatcher` 實作

### 資料流向

1. `svc.Watcher.Scan()` 掃描註冊路徑下的檔案狀態
2. 排程器 (Scheduler) 定期觸發 `collector.FlushHour(ctx)` 寫入統計資料至 `~/.config/file_watcher/stats/YYYY-MM-DDTHH.json`
3. 程式關閉時，`handler.finalFlush()` 排空警告、寫入最後統計並透過 `Notifier.Notify()` 發送最後報告

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

`svc.Collector` 使用 `sync.RWMutex` 保護其內部 map，確保併發安全。

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

- `config/` — 設定檔案載入與驗證
- `svc/` — 包含 watcher、collector、show (formatBytes, computeGrowth) 等服務單元測試
