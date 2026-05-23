# 全域設定重構驗證紀錄 (Global Config Refactoring Walkthrough)

我們已順利完成全域設定 (global config) 的重構，並簡化了相關函式之間的參數傳遞。

## 變更概述 (Changes Made)

1. 全域設定存取實作：
   - 在 `config/config.go` 中宣告了全域變數 `globalSettings`。
   - 新增了 `Get()` 取得函式。
   - 在 `Settings` 結構體中新增了 `StatsDir` 欄位。
   - 修改了 `Default()` 函式，若未設定 `stats_dir`，則會自動指派預設值，並在成功載入與驗證設定檔後將其更新至 `globalSettings`。

2. 簡化參數傳遞與相依注入：
   - 修改了 `handler/runner.go` 中的 `Wire` 函式，移除了 `homeDir` 參數，改為直接呼叫 `config.Get()` 取得 `cfg.StatsDir` 作為監控資料目錄，完全避免了參數傳遞。
   - 修改了 `cmd/start.go`，呼叫 `handler.Wire()` 時不再傳入任何參數。

3. 單元測試補強：
   - 在 `config/config_test.go` 中新增了 `TestGlobalConfig` 測試，驗證 `Get()` 能正常運作。

## 測試與驗證結果 (Testing & Verification)

1. 單元測試 (Unit Tests)：
   - 執行 `go test -v ./...`
   - 所有測試（包含新加入的 `TestGlobalConfig`）皆成功通過。

2. 編譯建置 (Build Verification)：
   - 執行 `go build -o file_watcher .`
   - 專案成功編譯，無編譯與語法錯誤。
