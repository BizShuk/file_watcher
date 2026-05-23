# File Watcher 搬移計畫 — 搬移 `notify` 至 `gosdk`

此變更旨在將原本位於 `file_watcher` 專案中的 `notify` 套件 (package) 完整搬移至共用的 `gosdk` 專案 (project)，以提升程式碼的重用性，並減少重複開發。

## 使用者審查要求 (User Review Required)

> [!IMPORTANT]
> 搬移 `notify` 套件將使 `file_watcher` 專案相依於 `gosdk` 中的 `notify` 實作。這要求 `gosdk` 與 `file_watcher` 的本機開發目錄結構保持相對關係，目前 `file_watcher` 的 `go.mod` 中已有 `replace github.com/bizshuk/gosdk => ../gosdk` 設定，可支援此變更。

> [!WARNING]
> 因為 `notify/slack.go` 依賴於 `github.com/slack-go/slack` 程式庫 (library)，搬移後我們需要在 `gosdk` 的 `go.mod` 中新增此相依套件。

## 開放性問題 (Open Questions)

無。

---

## 預期變更 (Proposed Changes)

### `gosdk` 元件 (gosdk component)

在 `gosdk` 中新增 `notify` 套件目錄及其實作檔案。

#### [NEW] [notifier.go](file:///Users/shuk/projects/gosdk/notify/notifier.go)
建立 `notify.Notifier` 介面 (interface)。

#### [NEW] [stdout.go](file:///Users/shuk/projects/gosdk/notify/stdout.go)
建立 `StdoutNotifier` 實作。

#### [NEW] [slack.go](file:///Users/shuk/projects/gosdk/notify/slack.go)
建立 `SlackNotifier` 實作。

#### [NEW] [multi.go](file:///Users/shuk/projects/gosdk/notify/multi.go)
建立 `Multi` 複合通知器實作。

#### [NEW] [notifier_test.go](file:///Users/shuk/projects/gosdk/notify/notifier_test.go)
建立通知器的單元測試 (unit test)。

#### [MODIFY] [go.mod](file:///Users/shuk/projects/gosdk/go.mod)
新增 `github.com/slack-go/slack` 套件依賴。

#### [MODIFY] [CLAUDE.md](file:///Users/shuk/projects/gosdk/CLAUDE.md)
更新 `gosdk` 專案結構描述，加入 `notify` 套件。

---

### `file_watcher` 元件 (file_watcher component)

自 `file_watcher` 專案移除 `notify` 目錄，並調整引用該套件的程式碼導入路徑 (import path)。

#### [DELETE] [multi.go](file:///Users/shuk/projects/file_watcher/notify/multi.go)
#### [DELETE] [notifier.go](file:///Users/shuk/projects/file_watcher/notify/notifier.go)
#### [DELETE] [notifier_test.go](file:///Users/shuk/projects/file_watcher/notify/notifier_test.go)
#### [DELETE] [slack.go](file:///Users/shuk/projects/file_watcher/notify/slack.go)
#### [DELETE] [stdout.go](file:///Users/shuk/projects/file_watcher/notify/stdout.go)

#### [MODIFY] [main_test.go](file:///Users/shuk/projects/file_watcher/main_test.go)
修改 `notify` 套件導入路徑為 `github.com/bizshuk/gosdk/notify`。

#### [MODIFY] [runner.go](file:///Users/shuk/projects/file_watcher/runner/runner.go)
修改 `notify` 套件導入路徑為 `github.com/bizshuk/gosdk/notify`。

#### [MODIFY] [go.mod](file:///Users/shuk/projects/file_watcher/go.mod)
執行 `go mod tidy` 移除不再直接使用的相依套件並更新模組相依。

#### [MODIFY] [CLAUDE.md](file:///Users/shuk/projects/file_watcher/CLAUDE.md)
更新 `file_watcher` 專案結構，移除 `notify` 套件說明。

---

## 驗證計畫 (Verification Plan)

### 自動化測試 (Automated Tests)

1. 在 `gosdk` 專案目錄下執行：
   `go test -v ./notify/...`
   確認新搬移的通知器測試均成功通過。

2. 在 `file_watcher` 專案目錄下執行：
   `go mod tidy`
   `go test -v ./...`
   確認測試無誤，且專案正常編譯。

### 手動驗證 (Manual Verification)

1. 編譯 `file_watcher` 二進位檔：
   `go build -o file_watcher .`
   確認可正常完成建置 (build)。
