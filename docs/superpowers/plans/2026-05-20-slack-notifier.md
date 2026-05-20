# Slack 機器人通知整合實作計畫 (Slack Bot Notification Integration Implementation Plan)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 實作 `SlackNotifier` 與 `MultiNotifier`，使檔案監控統計摘要能依據環境變數動態發送至 `Slack` 頻道中。

**Architecture:** 新增 `SlackNotifier` 呼叫 `github.com/slack-go/slack` SDK；實作 `MultiNotifier` 以包裝多個通知器；在 `main.go` 中讀取環境變數進行條件式配置。

**Tech Stack:** Go 1.20+, `github.com/slack-go/slack` SDK, 標準庫 (`net/http/httptest`).

---

### Task 1: 實作 MultiNotifier 複合通知器 (Implement MultiNotifier)

**Files:**
- Modify: `notifier.go`
- Create: `notifier_test.go`

- [ ] **Step 1: 在 notifier.go 中新增 MultiNotifier 結構**

在 `notifier.go` 中，實作 `MultiNotifier`，其 `Notify` 呼叫底下的所有通知器。

```go
package main

import (
	"fmt"
)

// Notifier defines how a stats summary is delivered.
type Notifier interface {
	Notify(summary string) error
}

// StdoutNotifier writes the summary to stdout.
type StdoutNotifier struct{}

// Notify implements Notifier by writing to stdout.
func (s *StdoutNotifier) Notify(summary string) error {
	fmt.Println(summary)
	return nil
}

// MultiNotifier composites multiple notifiers.
type MultiNotifier struct {
	notifiers []Notifier
}

// NewMultiNotifier creates a new MultiNotifier.
func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: notifiers}
}

// Notify implements Notifier by routing notifications to all registered notifiers.
func (m *MultiNotifier) Notify(summary string) error {
	var errs []error
	for _, n := range m.notifiers {
		if err := n.Notify(summary); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("some notifiers failed: %v", errs)
	}
	return nil
}
```

- [ ] **Step 2: 撰寫 MultiNotifier 的單元測試**

建立 `notifier_test.go`，驗證 `MultiNotifier` 能否成功觸發所有被持有的通知器。

```go
package main

import (
	"errors"
	"testing"
)

type mockNotifier struct {
	called bool
	err    error
}

func (m *mockNotifier) Notify(summary string) error {
	m.called = true
	return m.err
}

func TestMultiNotifier_Notify(t *testing.T) {
	n1 := &mockNotifier{}
	n2 := &mockNotifier{}

	multi := NewMultiNotifier(n1, n2)
	err := multi.Notify("test message")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !n1.called || !n2.called {
		t.Error("expected both notifiers to be called")
	}
}

func TestMultiNotifier_Notify_WithError(t *testing.T) {
	n1 := &mockNotifier{err: errors.New("fail")}
	n2 := &mockNotifier{}

	multi := NewMultiNotifier(n1, n2)
	err := multi.Notify("test message")
	if err == nil {
		t.Error("expected error, got nil")
	}

	if !n1.called || !n2.called {
		t.Error("expected both notifiers to be called even if one fails")
	}
}
```

- [ ] **Step 3: 執行測試驗證**

Run: `go test -v -run "TestMultiNotifier"`
Expected: PASS

- [ ] **Step 4: Commit Task 1**

Run:
```bash
git add notifier.go notifier_test.go
git commit -m "feat: implement MultiNotifier and its unit tests"
```

---

### Task 2: 實作 SlackNotifier 與 Mock 測試 (Implement SlackNotifier with SDK)

**Files:**
- Modify: `go.mod`
- Modify: `notifier.go`
- Modify: `notifier_test.go`

- [ ] **Step 1: 新增 Slack SDK 依賴**

在終端機安裝 `slack-go/slack` SDK：
Run: `go get github.com/slack-go/slack`
Expected: 成功下載並更新 `go.mod` 和 `go.sum`。

- [ ] **Step 2: 在 notifier.go 中新增 SlackNotifier 結構與 Notify 實作**

在 `notifier.go` 中，新增 `SlackNotifier` 並使用 SDK 來發送訊息：

```go
// Add package import at the top of notifier.go
// import (
//     ...
//     "github.com/slack-go/slack"
// )

type SlackNotifier struct {
	client    *slack.Client
	channelID string
}

func NewSlackNotifier(token, channelID string) *SlackNotifier {
	return &SlackNotifier{
		client:    slack.New(token),
		channelID: channelID,
	}
}

func (s *SlackNotifier) Notify(summary string) error {
	_, _, err := s.client.PostMessage(s.channelID, slack.MsgOptionText(summary, false))
	if err != nil {
		return fmt.Errorf("slack post message: %w", err)
	}
	return nil
}
```

注意：由於 `notifier.go` 需要引入 `github.com/slack-go/slack`，我們會在 `notifier.go` 頂部加入此 import。

- [ ] **Step 3: 撰寫 SlackNotifier 的 Mock 伺服器測試**

在 `notifier_test.go` 中新增 `TestSlackNotifier_Notify` 測試：

```go
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/slack-go/slack"
)

func TestSlackNotifier_Notify(t *testing.T) {
	// 建立 Mock HTTP 伺服器以攔截 Slack API 請求
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat.PostMessage" {
			t.Errorf("unexpected URL path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		// 返回 Slack 成功的回應結構
		json.NewEncoder(w).Encode(struct {
			OK bool `json:"ok"`
		}{OK: true})
	}))
	defer server.Close()

	// 初始化 SlackNotifier 並使用 Mock 伺服器位址作為 API 進入點
	notifier := NewSlackNotifier("fake-token", "fake-channel")
	notifier.client = slack.New("fake-token", slack.OptionAPIURL(server.URL+"/"))

	err := notifier.Notify("hello from test")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
```

- [ ] **Step 4: 執行單元測試**

Run: `go test -v -run "TestSlackNotifier"`
Expected: PASS

- [ ] **Step 5: Commit Task 2**

Run:
```bash
git add go.mod go.sum notifier.go notifier_test.go
git commit -m "feat: implement SlackNotifier using slack-go SDK with mock server test"
```

---

### Task 3: 在 main.go 中整合與驗證 (Integrate in main.go and verify)

**Files:**
- Modify: `main.go`

- [ ] **Step 1: 修改 main.go 以進行條件式配置**

在 `main.go` 的 `main` 函式中，偵測環境變數並配置 `MultiNotifier`：

```go
	// 建立通知器列表
	var notifiers []Notifier
	notifiers = append(notifiers, &StdoutNotifier{})

	// 檢查 Slack 環境變數
	slackToken := os.Getenv("SLACK_BOT_TOKEN")
	slackChannel := os.Getenv("SLACK_CHANNEL_ID")
	if slackToken != "" && slackChannel != "" {
		log.Info("Slack notification enabled", "channel", slackChannel)
		notifiers = append(notifiers, NewSlackNotifier(slackToken, slackChannel))
	}

	notifier := NewMultiNotifier(notifiers...)
```

- [ ] **Step 2: 重新編譯專案**

Run: `go build -o file_watcher .`
Expected: 編譯成功。

- [ ] **Step 3: 執行所有單元測試**

Run: `go test -v ./...`
Expected: PASS 所有測試 (包含我們先前實作的單元測試)。

- [ ] **Step 4: Git 提交所有修改並結案**

Run:
```bash
git add main.go
git commit -m "feat: integrate SlackNotifier with composite MultiNotifier in main entry point"
```
