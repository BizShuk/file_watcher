# 設定匯出子命令設計規格書 (Settings Exporter Command Design Spec)

這份文件定義了在檔案監控服務 (file watcher service) 中新增 `export` 子命令 (subcommand) 的設計。

## 背景與動機 (Background and Motivation)

使用者需要一個能夠讀取目前系統的設定檔 `settings.json` 並將其格式化輸出的命令列工具。這有助於備份、偵錯或將設定傳遞給其他腳本或系統。

## 設計細節 (Design Details)

### 1. 命令列語法 (CLI Syntax)

使用者可以透過以下指令來執行設定匯出：

`./file_watcher export`

### 2. 核心邏輯 (Core Logic)

在獨立的 Go 原始碼檔案 `export.go` 中，實作 `runExport(w io.Writer) error` 函式：

1. 呼叫 `Load()` 載入 `Settings`。`Load()` 會自動尋找並讀取預設路徑 `~/.config/file_watcher/settings.json`。如果設定檔不存在，則會自動從嵌入的 `settings.default.json` 建立一個新的設定檔，再進行載入。
2. 使用 `json.MarshalIndent` 將載入的 `Settings` 結構序列化為具備縮排排版的美化 `JSON` 格式。
3. 將序列化後的資料寫入傳入的 `w` (在主程式中為 `os.Stdout`)，並在末尾寫入換行符。

### 3. 主程式整合 (Integration with main.go)

在 `main.go` 中，解析第一個參數是否為 `export`，如果是則執行匯出邏輯：

```go
case "export":
    if err := runExport(os.Stdout); err != nil {
        fmt.Fprintf(os.Stderr, "export: %v\n", err)
        os.Exit(1)
    }
    return
```

## 單元測試計畫 (Unit Testing Plan)

我們將在 `export_test.go` 中加入單元測試：

1. 使用 Go 測試框架的 `t.TempDir()` 建立一個獨立的暫時目錄。
2. 暫時將 `homeDirFn` 變數指向該暫時目錄，確保測試不會影響或讀取到實際使用者的 `~/.config/file_watcher/settings.json` 檔案。
3. 呼叫 `runExport` 並傳入 `bytes.Buffer` 作為輸出目標。
4. 驗證輸出是否為合法的 `JSON`，且其內容與預設設定相符。
5. 在測試結束後，將 `homeDirFn` 還原。
