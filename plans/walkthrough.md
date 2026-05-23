# 重構與無效程式碼清理驗證紀錄 (Refactoring & Dead Code Cleanup Walkthrough)

我們已順利完成套件結構的合併與重新命名重構，並清理了專案中被偵測到的無效程式碼 (dead code)。

## 變更概述 (Changes Made)

1. `svc` 套件合併：
   - 移除了原有的 `show/`, `stats/`, `warning/`, `watcher/` 目錄。
   - 將這些目錄下的所有 Go 原始碼檔與測試檔案合併移入新建的 `svc/` 目錄中，並統一宣告為 `package svc`。
   - 刪除 `show.go` 中與 `entry.go` 衝突的 `StatFile` 結構定義，並移除了對舊套件的內部引用，將代碼內的 `stats.Entry` 簡化為 `Entry`。

2. `handler` 套件移動：
   - 移了原有的 `runner/` 目錄。
   - 建立新建的 `handler/` 目錄，將生命週期管理主控程式移至 `handler/runner.go`，並統一宣告為 `package handler`。
   - 更新引用的內部服務路徑至 `github.com/shuk/file_watcher/svc`。

3. 無效程式碼 (Dead Code) 清理：
   - 偵測到 `svc/collector.go` 中的 `Collector.AddOrUpdate` 與 `Collector.Remove`，以及 `svc/sink.go` 中的 `Sink.Add` 未在主程式碼流程中被呼叫使用。
   - 這些方法起初用於 fsnotify 即時檔案系統事件監聽處理，而在重構為排程主動掃描模式後已不再使用。
   - 我們已將這些方法以及對應定義的 `Recorder` 與 `SinkInterface` 介面安全刪除。
   - 更新了 `svc/stats_test.go`，移除已刪除方法的測試，並調整剩餘測試使其直接將 Mock 資料寫入 `c.data` 中進行驗證。

4. 命令列入口與文檔更新：
   - 更新了 `cmd/show.go` 與 `cmd/start.go` 的 package import，分別指向 `svc` 與 `handler`。
   - 更新了 `CLAUDE.md` 與 `docs/PROJECT_SUMMARY.md` 的專案目錄結構說明與架構描述，並遵循全域規則，將所有 `**` 粗體語法替換為 `backticks` 或一般文字。

## 測試與驗證結果 (Testing & Verification)

1. `單元測試 (Unit Tests)`：
   - 執行 `go test ./...`
   - 結果：`ok github.com/shuk/file_watcher/svc` ＆ `ok github.com/shuk/file_watcher/config`，所有單元測試皆順利通過。

2. `編譯建置 (Build Verification)`：
   - 執行 `go build -o file_watcher .`
   - 結果：順利編譯出二進位檔，無任何編譯或語法錯誤。

3. `無效程式碼檢測 (Dead Code Detection Verification)`：
   - 再次執行 `deadcode ./...`
   - 結果：輸出為空，已無任何無法抵達/未使用到的 Go 函數。
