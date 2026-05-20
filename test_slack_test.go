package main

import (
	"errors"
	"strings"
	"testing"
)

type mockTestNotifier struct {
	token     string
	channelID string
	err       error
	msgSent   string
}

func (m *mockTestNotifier) Notify(summary string) error {
	m.msgSent = summary
	return m.err
}

func TestRunTestSlack_MissingToken(t *testing.T) {
	// Ensure env variables are cleared/set for test
	t.Setenv("SLACK_BOT_TOKEN", "")
	t.Setenv("SLACK_CHANNEL_ID", "fake-channel")

	err := runTestSlack()
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

	err := runTestSlack()
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

	// Mock newSlackNotifierFn
	origFn := newSlackNotifierFn
	defer func() { newSlackNotifierFn = origFn }()

	mockNotifier := &mockTestNotifier{
		token:     "fake-token",
		channelID: "fake-channel",
	}
	newSlackNotifierFn = func(token, channelID string) Notifier {
		if token != "fake-token" || channelID != "fake-channel" {
			t.Errorf("unexpected params: token=%s, channelID=%s", token, channelID)
		}
		return mockNotifier
	}

	err := runTestSlack()
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

	// Mock newSlackNotifierFn
	origFn := newSlackNotifierFn
	defer func() { newSlackNotifierFn = origFn }()

	mockNotifier := &mockTestNotifier{
		token:     "fake-token",
		channelID: "fake-channel",
		err:       errors.New("slack post failed"),
	}
	newSlackNotifierFn = func(token, channelID string) Notifier {
		return mockNotifier
	}

	err := runTestSlack()
	if err == nil {
		t.Error("expected error from notifier.Notify, got nil")
	}
	if !strings.Contains(err.Error(), "slack post failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}
