package scheduler

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/shuk/file_watcher/notify"
	"github.com/shuk/file_watcher/stats"
	"github.com/shuk/file_watcher/warning"
)

// Scheduler periodically flushes stats and notifies via Notifier.
type scheduler struct {
	collector     stats.Flusher
	warnings     *warning.Sink
	notifier      notify.Notifier
	batchPeriod   time.Duration
	retentionDays int
}

// New creates a new scheduler.
func New(
	collector stats.Flusher,
	warnings *warning.Sink,
	notifier notify.Notifier,
	period time.Duration,
	retentionDays int,
) *scheduler {
	return &scheduler{
		collector:     collector,
		warnings:      warnings,
		notifier:      notifier,
		batchPeriod:   period,
		retentionDays: retentionDays,
	}
}

// Start begins the periodic flush loop.
func (s *scheduler) Start(ctx context.Context) error {
	ticker := time.NewTicker(s.batchPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.flush(ctx)
		case <-ctx.Done():
			s.flush(ctx)
			return ctx.Err()
		}
	}
}

func (s *scheduler) flush(ctx context.Context) {
	warnings := s.warnings.Drain()

	if err := s.collector.FlushHour(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "flush error: %v\n", err)
	}
	s.collector.Clear()

	if err := s.collector.Prune(ctx, s.retentionDays); err != nil {
		fmt.Fprintf(os.Stderr, "prune error: %v\n", err)
	}

	message := fmt.Sprintf("[%s] Stats flushed and pruned", time.Now().Format(time.RFC3339))
	if len(warnings) > 0 {
		message += "\n\nWarnings during file monitoring:"
		for _, w := range warnings {
			message += fmt.Sprintf("\n- %s", w)
		}
	}

	if err := s.notifier.Notify(ctx, message); err != nil {
		fmt.Fprintf(os.Stderr, "notify error: %v\n", err)
	}
}

// FlushNow triggers an immediate flush.
func (s *scheduler) FlushNow() {
	s.flush(context.Background())
}