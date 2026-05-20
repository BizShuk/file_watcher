# Slack 機器人通知器整合設計規格書 (Slack Bot Notifier Integration Design Spec)

這份文件定義了在檔案監控服務 (file watcher service) 中整合 `Slack Bot SDK`，實現將統計摘要同步發送至 `Slack` 頻道的設計。

## 背景與動機 (Background and Motivation)

當前服務僅支援將統計摘要輸出至標準輸出 (`StdoutNotifier`)。為方便使用者遠端監控，本設計引入 `Slack Bot` 通知功能。使用者只需將機器人權杖 (Bot Token) 與頻道識別碼 (Channel ID) 以環境變數傳入，服務即會啟用 `Slack` 通知功能。

## 設計細節 (Design Details)

### 1. 通知器架構擴充 (Notifier Architecture Expansion)

我們將引入多重通知器 (MultiNotifier) 與 `Slack` 通知器 (SlackNotifier)：

- **MultiNotifier**：實作 `Notifier` 介面，內部持有一個通知器陣列 (`[]Notifier`)。當觸發 `Notify` 時，會依序呼叫陣列中所有的通知器。這可確保在不破壞現有 `StdoutNotifier` 功能的前提下，動態啟用多個通知通道。
- **SlackNotifier**：實作 `Notifier` 介面，使用官方 SDK `github.com/slack-go/slack` 進行 API 呼叫。

### 2. 環境變數整合 (Environment Variables)

服務啟動時會自動偵測以下兩個環境變數：

- `SLACK_BOT_TOKEN`：`Slack` 機器人權杖（以 `xoxb-` 開頭）
- `SLACK_CHANNEL_ID`：發送通知的目標頻道 ID（例如 `C12345678`）

當這兩個環境變數同時存在時，程式將自動初始化 `SlackNotifier` 並將其加入 `MultiNotifier` 中。

### 3. API 整合方式 (API Integration)

使用 `github.com/slack-go/slack` SDK 呼叫 `chat.PostMessage` 端點。

## 單元測試計畫 (Unit Testing Plan)

我們將在 `notifier_test.go` 中加入單元測試：

1. **TestSlackNotifier_Notify_Mock**：使用 `httptest.NewServer` 模擬 `Slack` 的 API 伺服器，驗證 `SlackNotifier` 是否能正確封裝並傳送 JSON 訊息。
2. **TestMultiNotifier_Notify**：驗證 `MultiNotifier` 是否能正常依序觸發底下的所有通知器。
