# File Watcher

一個用 Go 撰寫的檔案監控服務，可監控目錄變更並定期將檔案統計資料寫入磁碟。

## 功能

- **即時監控**：使用 `fsnotify` 監控指定目錄的檔案變更（建立、修改、刪除）
- **統計收集**：收集每個檔案的大小與最後修改時間
- **定期寫出**：每個 `batchPeriod`（預設 1 小時）將統計資料寫入 `~/.config/file_watcher/stats/YYYY-MM-DDTHH.json`
- **保留管理**：自動刪除超過 `stats_retention_days`（預設 7 天）的舊統計檔案
- **增長報告**：透過 `show` 子命令以橫條圖顯示檔案大小變化

## 安裝

```bash
go build -o file_watcher .
```

## 使用方式

### 啟動監控服務

```bash
./file_watcher
```

服務會讀取 `~/.config/file_watcher/settings.json` 的設定並開始監控。

### 查看磁碟使用量增長

```bash
./file_watcher show
```

輸出範例：

```
磁碟使用量增長報告（初始 vs 最新）
================================================================================

/tmp/log.txt          ████████████████████████████████████████  2.5MB (+150%)
/tmp/data.json        ████████████████████████                  1.2MB (+80%)
/tmp/new.dat          █████████████████████                    (NEW)
```

### 匯出目前設定檔內容

```bash
./file_watcher export
```

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
|------|------|------|
| `watch_list` | `[]string` | 要監控的目錄或檔案路徑（支援使用 `~` 代表家目錄） |
| `exclude_list` | `[]string` | 要排除的副檔名（例如 `.git`） |
| `batch_period` | `string` | 統計寫出的間隔時間（例如 `1h`、`30m`） |
| `stats_retention_days` | `int` | 統計檔案的保留天數 |

## 架構

```
main.go          # 進入點，初始化所有元件並處理子命令
  ├── config.go       # 讀取並驗證設定檔
  ├── export.go       # 匯出設定檔為格式化的 JSON 輸出
  ├── watcher.go      # fsnotify 包裝器，處理檔案事件
  ├── stats.go        # StatsCollector 介面與實作
  ├── scheduler.go    # 定期執行 flush 與 prune
  └── notifier.go     # 通知介面（目前為 stdout）
```

### 核心介面（ISP + DIP）

- **StatsCollector**：`AddOrUpdate`、`Remove`、`FlushHour`、`Prune`、`Clear`
- **Notifier**：`Notify(summary string) error`
- **WatcherOps**：`Add`、`Start`、`Close`

### 資料流向

1. `fsWatcher.Start()` 監聽 fsnotify 事件並呼叫 handler
2. Handler 呼叫 `collector.AddOrUpdate(path, size, modTime)` 記錄檔案變更
3. `Scheduler.run()` 每隔 `batchPeriod` 呼叫 `flush()`
4. `flush()` 將統計寫入 `~/.config/file_watcher/stats/YYYY-MM-DDTHH.json`
5. `Notifier.Notify()` 輸出摘要到標準輸出

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