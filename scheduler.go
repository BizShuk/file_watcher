package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Scheduler periodically flushes stats and notifies via Notifier.
type Scheduler struct {
	collector     StatsCollector
	notifier      Notifier
	batchPeriod   time.Duration
	retentionDays int
	ticker        *time.Ticker
	stop          chan struct{}
	once          sync.Once
}

// NewScheduler creates a new Scheduler.
func NewScheduler(col StatsCollector, notif Notifier, period time.Duration, retentionDays int) *Scheduler {
	return &Scheduler{
		collector:     col,
		notifier:      notif,
		batchPeriod:   period,
		retentionDays: retentionDays,
		stop:          make(chan struct{}),
	}
}

// Start begins the periodic flush loop.
// (DIP: depends on StatsCollector + Notifier interfaces, not concrete types)
func (s *Scheduler) Start() error {
	s.ticker = time.NewTicker(s.batchPeriod)
	go s.run()
	return nil
}

func (s *Scheduler) run() {
	for {
		select {
		case <-s.ticker.C:
			s.flush()
		case <-s.stop:
			s.ticker.Stop()
			return
		}
	}
}

func (s *Scheduler) flush() {
	// Retrieve warnings before clearing collector.
	warnings := s.collector.GetWarnings()

	if err := s.collector.FlushHour(); err != nil {
		fmt.Fprintf(os.Stderr, "flush error: %v\n", err)
		return
	}
	s.collector.Clear()

	// Prune old files.
	if err := s.collector.Prune(s.retentionDays); err != nil {
		fmt.Fprintf(os.Stderr, "prune error: %v\n", err)
	}

	message := fmt.Sprintf("[%s] Stats flushed and pruned", time.Now().Format(time.RFC3339))
	if len(warnings) > 0 {
		message += "\n\n⚠️ Warnings during file monitoring:"
		for _, w := range warnings {
			message += fmt.Sprintf("\n- %s", w)
		}
	}
	s.collector.ClearWarnings()

	if err := s.notifier.Notify(message); err != nil {
		fmt.Fprintf(os.Stderr, "notify error: %v\n", err)
	}
}

// FlushNow flushes immediately and stops the scheduler.
// Called on SIGTERM/SIGINT for graceful shutdown.
func (s *Scheduler) FlushNow() {
	s.once.Do(func() {
		close(s.stop)
		if s.ticker != nil {
			s.ticker.Stop()
		}
		s.flush()
	})
}
