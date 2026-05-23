package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/shuk/file_watcher/notify"
)

func TestSlackIntegration(t *testing.T) {
	slackToken := os.Getenv("SLACK_BOT_TOKEN")
	slackChannel := os.Getenv("SLACK_CHANNEL_ID")

	if slackToken == "" {
		t.Skip("SLACK_BOT_TOKEN is not set, skipping Slack integration test")
	}
	if slackChannel == "" {
		t.Skip("SLACK_CHANNEL_ID is not set, skipping Slack integration test")
	}

	notifier := notify.NewSlackNotifier(slackToken, slackChannel)
	msg := fmt.Sprintf("Slack 整合測試訊息 (Slack Integration Test Message) - 發送時間: %s", time.Now().Format(time.RFC3339))

	t.Logf("正在發送測試訊息至頻道 %s (Sending test message to channel %s)...", slackChannel, slackChannel)
	if err := notifier.Notify(context.Background(), msg); err != nil {
		t.Fatalf("發送測試訊息失敗 (Failed to send test message): %v", err)
	}

	t.Log("測試訊息發送成功 (Test message sent successfully)!")
}
