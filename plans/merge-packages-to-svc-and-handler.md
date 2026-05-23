# 合併與重新整理套件結構 (Merge and Refactor Package Structure)

本計劃旨在重構專案的套件結構，將四個小型服務套件（`show`、`stats`、`warning`、`watcher`）合併為一個統一的 `svc` 套件 (service package)，並將負責驅動生命週期的 `runner` 套件移動至 `handler` 套件中。

## 用戶審查要求 (User Review Required)

這是一個較大的架構調整，將修改多個 Go 檔案的套件宣告 (package declaration) 與引入路徑 (import paths)，並移除原本已無用的舊目錄。請確認此結構調整符合專案未來的擴充規劃。

## 待解決問題 (Open Questions)

無特別設計疑慮。我們將確保合併後的單元測試 (unit tests) 與編譯能完全正常運作。

## 建議的變更 (Proposed Changes)

---

### 套件合併：`show`、`stats` 、`warning`、`watcher` -> `svc`

將前述四個目錄下的所有程式碼檔案與測試檔案移至 `svc` 目錄，並將套件宣告統一改為 `package svc`。同時更新彼此之間的內部呼叫（例如原本的 `stats.Entry` 在同一套件下可直接使用 `Entry`）。

#### [NEW] [growth.go](file:///Users/shuk/projects/file_watcher/svc/growth.go)

#### [NEW] [show.go](file:///Users/shuk/projects/file_watcher/svc/show.go)

#### [NEW] [show_test.go](file:///Users/shuk/projects/file_watcher/svc/show_test.go)

#### [NEW] [collector.go](file:///Users/shuk/projects/file_watcher/svc/collector.go)

#### [NEW] [entry.go](file:///Users/shuk/projects/file_watcher/svc/entry.go)

#### [NEW] [stats_test.go](file:///Users/shuk/projects/file_watcher/svc/stats_test.go)

#### [NEW] [sink.go](file:///Users/shuk/projects/file_watcher/svc/sink.go)

#### [NEW] [watcher.go](file:///Users/shuk/projects/file_watcher/svc/watcher.go)

#### [NEW] [watcher_test.go](file:///Users/shuk/projects/file_watcher/svc/watcher_test.go)

#### [DELETE] [growth.go](file:///Users/shuk/projects/file_watcher/show/growth.go)

#### [DELETE] [show.go](file:///Users/shuk/projects/file_watcher/show/show.go)

#### [DELETE] [show_test.go](file:///Users/shuk/projects/file_watcher/show/show_test.go)

#### [DELETE] [collector.go](file:///Users/shuk/projects/file_watcher/stats/collector.go)

#### [DELETE] [entry.go](file:///Users/shuk/projects/file_watcher/stats/entry.go)

#### [DELETE] [stats_test.go](file:///Users/shuk/projects/file_watcher/stats/stats_test.go)

#### [DELETE] [sink.go](file:///Users/shuk/projects/file_watcher/warning/sink.go)

#### [DELETE] [watcher.go](file:///Users/shuk/projects/file_watcher/watcher/watcher.go)

#### [DELETE] [watcher_test.go](file:///Users/shuk/projects/file_watcher/watcher/watcher_test.go)

---

### 生命週期管理移動：`runner` -> `handler`

將 `runner/runner.go` 移至 `handler/runner.go`，並將套件宣告改為 `package handler`。更新其內部所使用的 `svc` 依賴路徑與相關結構名稱。

#### [NEW] [runner.go](file:///Users/shuk/projects/file_watcher/handler/runner.go)

#### [DELETE] [runner.go](file:///Users/shuk/projects/file_watcher/runner/runner.go)

---

### 指令進入點與設定檔調整

更新命令列定義檔案，使其引入新的 `svc` 與 `handler` 套件，並替換相關呼叫。

#### [MODIFY] [show.go](file:///Users/shuk/projects/file_watcher/cmd/show.go)

#### [MODIFY] [start.go](file:///Users/shuk/projects/file_watcher/cmd/start.go)

#### [MODIFY] [config.go](file:///Users/shuk/projects/file_watcher/config/config.go)

#### [MODIFY] [PROJECT_SUMMARY.md](file:///Users/shuk/projects/file_watcher/docs/PROJECT_SUMMARY.md)

#### [MODIFY] [CLAUDE.md](file:///Users/shuk/projects/file_watcher/CLAUDE.md)

---

## 驗證計劃 (Verification Plan)

### 自動化測試

- 執行 `go test ./...` 確保所有移至 `svc` 中的單元測試依然通過。
- 執行 `go build -o file_watcher .` 確保專案能正常編譯。

### 手動驗證

更新命令列定義檔案，使其引入新的 `svc` 與 `handler` 套件，並替換相關呼叫。

#### [MODIFY] [show.go](file:///Users/shuk/projects/file_watcher/cmd/show.go)

#### [MODIFY] [start.go](file:///Users/shuk/projects/file_watcher/cmd/start.go)

#### [MODIFY] [config.go](file:///Users/shuk/projects/file_watcher/config/config.go)

#### [MODIFY] [PROJECT_SUMMARY.md](file:///Users/shuk/projects/file_watcher/docs/PROJECT_SUMMARY.md)

#### [MODIFY] [CLAUDE.md](file:///Users/shuk/projects/file_watcher/CLAUDE.md)

---

## 驗證計劃 (Verification Plan)

### 自動化測試

- 執行 `go test ./...` 確保所有移至 `svc` 中的單元測試依然通過。
- 執行 `go build -o file_watcher .` 確保專案能正常編譯。

### 手動驗證

- 執行 `./file_watcher` (預設為 `show` 邏輯) 驗證圖表是否能正常輸出。
- 執行 `./file_watcher start` 驗證監控程式是否能正常啟動。
