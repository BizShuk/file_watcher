# 實作全域設定 (Global Config) 系統與簡化參數傳遞計畫

此計畫旨在為 `file_watcher` 專案引入全域設定 (global config) 機制，使設定載入後能儲存於全域變數，並透過取得方法 (get method) 在需要的地方存取設定。如此一來，即可避免將設定結構體 (Settings struct) 作為參數在多處（例如 `handler.Wire` 等函式）之間層層傳遞，從而簡化程式碼結構與相依注入 (dependency injection) 的實作。

## 使用者審查要求 (User Review Required)

> [!IMPORTANT]
> 引入全域設定後，`handler.Wire` 將不再需要 `cfg *config.Settings` 參數，這是一個破壞性變更 (breaking change)。所有呼叫 `handler.Wire` 的地方（例如 `cmd/start.go`）都必須同步調整。

## 開放性問題 (Open Questions)

無。

---

## 預期變更 (Proposed Changes)

### `config` 套件 (config package)

在設定套件中加入全域變數、取得與設定全域變數的方法，並在載入設定時自動存入全域變數。

#### [MODIFY] [config.go](file:///Users/shuk/projects/file_watcher/config/config.go)
- 宣告全域變數 `globalSettings *Settings`。
- 新增 `Get()` 函式，用於取得全域設定。
- 新增 `Set(cfg *Settings)` 函式，用於測試或手動設定時更新全域變數。
- 修改 `Default()` 函式，在成功載入並驗證設定後，將結果指派給 `globalSettings`。

---

### `handler` 套件 (handler package)

修改執行器 (runner) 的相依注入邏輯，直接從全域設定取得所需設定。

#### [MODIFY] [runner.go](file:///Users/shuk/projects/file_watcher/handler/runner.go)
- 修改 `Wire(homeDir string, cfg *config.Settings)` 簽名為 `Wire(homeDir string)`。
- 在 `Wire` 內部透過 `config.Get()` 取得設定結構體。

---

### `cmd` 套件 (cmd package)

更新啟動命令的相依注入呼叫方式。

#### [MODIFY] [start.go](file:///Users/shuk/projects/file_watcher/cmd/start.go)
- 修改呼叫 `handler.Wire(homeDir, cfg)` 的程式碼，移除 `cfg` 參數，改為直接呼叫 `handler.Wire(homeDir)`。

---

## 驗證計畫 (Verification Plan)

### 自動化測試 (Automated Tests)
- 在專案根目錄下執行單元測試：
  `go test -v ./...`
  確認所有現有的測試以及新增的測試均能編譯並成功通過。

### 手動驗證 (Manual Verification)
- 測試編譯：
  `go build -o file_watcher .`
  確認二進位檔能正常建置，無編譯錯誤。
- 啟動監聽器測試：
  `./file_watcher start`
  確認程式可正常啟動並載入全域設定。
