package main

import (
	"errors"
	"fmt"
	"os"
	"time"
)

var newSlackNotifierFn = func(token, channelID string) Notifier {
	return NewSlackNotifier(token, channelID)
}

// runTestSlack runs the test-slack subcommand.
func runTestSlack() error {
	slackToken := os.Getenv("SLACK_BOT_TOKEN")
	slackChannel := os.Getenv("SLACK_CHANNEL_ID")

	if slackToken == "" {
		return errors.New("SLACK_BOT_TOKEN is not set")
	}
	if slackChannel == "" {
		return errors.New("SLACK_CHANNEL_ID is not set")
	}

	notifier := newSlackNotifierFn(slackToken, slackChannel)
	msg := fmt.Sprintf("Slack 整合測試訊息 (Slack Integration Test Message) - 發送時間: %s", time.Now().Format(time.RFC3339))

	fmt.Printf("正在發送測試訊息至頻道 %s (Sending test message to channel %s)...\n", slackChannel, slackChannel)
	if err := notifier.Notify(msg); err != nil {
		return err
	}

	fmt.Println("測試訊息發送成功 (Test message sent successfully)!")
	return nil
}
