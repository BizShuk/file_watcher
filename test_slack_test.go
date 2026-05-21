package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/shuk/file_watcher/notify"
)

type mockTestNotifier struct {
	token     string
	channelID string
	err       error
	msgSent   string
}

func (m *mockTestNotifier) Notify(ctx context.Context, summary string) error {
	m.msgSent = summary
	return m.err
}

func TestRunTestSlack_MissingToken(t *testing.T) {
	t.Setenv("SLACK_BOT_TOKEN", "")
	t.Setenv("SLACK_CHANNEL_ID", "fake-channel")

	err := runTestSlack(func(token, channelID string) notify.Notifier {
		return notify.NewSlackNotifier(token, channelID)
	})
	if err == nil {
		t.Error("expected error due to missing SLACK_BOT_TOKEN, got nil")
	}
	if !strings.Contains(err.Error(), "SLACK_BOT_TOKEN is not set") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunTestSlack_MissingChannel(t *testing.T) {
	t.Setenv("SLACK_BOT_TOKEN", "fake-token")
	t.Setenv("SLACK_CHANNEL_ID", "")

	err := runTestSlack(func(token, channelID string) notify.Notifier {
		return notify.NewSlackNotifier(token, channelID)
	})
	if err == nil {
		t.Error("expected error due to missing SLACK_CHANNEL_ID, got nil")
	}
	if !strings.Contains(err.Error(), "SLACK_CHANNEL_ID is not set") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunTestSlack_Success(t *testing.T) {
	t.Setenv("SLACK_BOT_TOKEN", "fake-token")
	t.Setenv("SLACK_CHANNEL_ID", "fake-channel")

	mockNotifier := &mockTestNotifier{
		token:     "fake-token",
		channelID: "fake-channel",
	}

	err := runTestSlack(func(token, channelID string) notify.Notifier {
		if token != "fake-token" || channelID != "fake-channel" {
			t.Errorf("unexpected params: token=%s, channelID=%s", token, channelID)
		}
		return mockNotifier
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(mockNotifier.msgSent, "Slack 整合測試訊息") {
		t.Errorf("unexpected message sent: %s", mockNotifier.msgSent)
	}
}

func TestRunTestSlack_Fail(t *testing.T) {
	t.Setenv("SLACK_BOT_TOKEN", "fake-token")
	t.Setenv("SLACK_CHANNEL_ID", "fake-channel")

	mockNotifier := &mockTestNotifier{
		token:     "fake-token",
		channelID: "fake-channel",
		err:       errors.New("slack post failed"),
	}

	err := runTestSlack(func(token, channelID string) notify.Notifier {
		return mockNotifier
	})
	if err == nil {
		t.Error("expected error from notifier.Notify, got nil")
	}
	if !strings.Contains(err.Error(), "slack post failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}
