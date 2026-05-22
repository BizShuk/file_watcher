package scheduler

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/shuk/file_watcher/notify"
	"github.com/shuk/file_watcher/stats"
	"github.com/shuk/file_watcher/warning"
)

// job represents a scheduled task with its own interval.
type job struct {
	name     string
	interval time.Duration
	fn       func(context.Context) error
}

// Scheduler periodically flushes stats and notifies via Notifier.
type scheduler struct {
	collector stats.Flusher
	warnings  *warning.Sink
	notifier  notify.Notifier
	jobs      []job
	mu        sync.Mutex
}

// New creates a new scheduler.
func New(
	collector stats.Flusher,
	warnings *warning.Sink,
	notifier notify.Notifier,
) *scheduler {
	return &scheduler{
		collector: collector,
		warnings:  warnings,
		notifier:  notifier,
	}
}

// Every schedules a job to run at the specified interval.
// The job function receives a context and returns an error on failure.
func (s *scheduler) Every(name string, interval time.Duration, fn func(context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = append(s.jobs, job{name: name, interval: interval, fn: fn})
}

// Start runs all scheduled jobs concurrently.
// It blocks until the context is cancelled.
func (s *scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	jobs := make([]job, len(s.jobs))
	copy(jobs, s.jobs)
	s.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup

	for _, j := range jobs {
		wg.Add(1)
		go func(j job) {
			defer wg.Done()
			ticker := time.NewTicker(j.interval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := j.fn(ctx); err != nil {
						fmt.Fprintf(os.Stderr, "scheduler job %q error: %v\n", j.name, err)
					}
				case <-ctx.Done():
					return
				}
			}
		}(j)
	}

	<-ctx.Done()
	cancel()
	wg.Wait()
	return ctx.Err()
}

// FlushNow triggers an immediate flush of all stats and sends a notification.
// This is kept for backward compatibility.
func (s *scheduler) FlushNow() {
	ctx := context.Background()
	var warnings []string
	if s.warnings != nil {
		warnings = s.warnings.Drain()
	}

	if err := s.collector.FlushHour(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "flush error: %v\n", err)
	}
	s.collector.Clear()

	if err := s.collector.Prune(ctx, 7); err != nil {
		fmt.Fprintf(os.Stderr, "prune error: %v\n", err)
	}

	message := fmt.Sprintf("[%s] Stats flushed and pruned", time.Now().Format(time.RFC3339))
	if len(warnings) > 0 {
		message += "\n\nWarnings during file monitoring:"
		var b strings.Builder
		for _, w := range warnings {
			b.WriteString("\n- ")
			b.WriteString(w)
		}
		message += b.String()
	}

	if err := s.notifier.Notify(ctx, message); err != nil {
		fmt.Fprintf(os.Stderr, "notify error: %v\n", err)
	}
}