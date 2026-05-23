# 全域設定重構工作清單 (Global Config Refactoring Task Checklist)

- `[x]` 在 `config/config.go` 中實作全域變數與存取方法
  - `[x]` 新增 `globalSettings` 變數
  - `[x]` 實作 `Get()` 與 `Set()` 方法
  - `[x]` 修改 `Default()` 以自動更新 `globalSettings`
- `[x]` 修改 `handler/runner.go` 調整相依注入簽名
  - `[x]` 修改 `Wire` 函式簽名，移除 `cfg` 參數
  - `[x]` 在 `Wire` 內部透過 `config.Get()` 取得設定
- `[x]` 更新 `cmd/start.go` 中的呼叫方式
  - `[x]` 移除 `handler.Wire` 呼叫時的 `cfg` 參數
- `[x]` 執行驗證
  - `[x]` 執行 `go test ./...` 驗證測試是否全部通過
  - `[x]` 執行 `go build -o file_watcher .` 驗證是否編譯成功
