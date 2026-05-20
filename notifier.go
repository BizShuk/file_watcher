package main

import (
	"fmt"

	"github.com/slack-go/slack"
)

// Notifier defines how a stats summary is delivered.
// (ISP: single-method interface, easy to implement new variants)
// (DIP: scheduler depends on this abstraction, not concrete output)
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

// SlackNotifier delivers statistics summaries to a Slack channel via the Slack Bot SDK.
type SlackNotifier struct {
	client    *slack.Client
	channelID string
}

// NewSlackNotifier creates a new SlackNotifier configured with a Bot Token and target Channel ID.
func NewSlackNotifier(token, channelID string) *SlackNotifier {
	return &SlackNotifier{
		client:    slack.New(token),
		channelID: channelID,
	}
}

// Notify implements Notifier by posting the summary text to the specified Slack channel.
func (s *SlackNotifier) Notify(summary string) error {
	_, _, err := s.client.PostMessage(s.channelID, slack.MsgOptionText(summary, false))
	if err != nil {
		return fmt.Errorf("slack post message: %w", err)
	}
	return nil
}
