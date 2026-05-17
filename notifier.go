package main

import (
	"fmt"
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