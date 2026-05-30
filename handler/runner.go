package handler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bizshuk/file_watcher/config"
	"github.com/bizshuk/file_watcher/svc"
	"github.com/bizshuk/gosdk/scheduler"
)

// runtime holds the application components started together.
type runtime struct {
	watcher   svc.Watcher
	collector *svc.Collector
	sched     *scheduler.Scheduler
}

// Wire builds the runtime from configuration. It is the sole DI entry point.
func Wire() (*runtime, error) {
	cfg := config.GlobalSettings
	w, err := svc.NewWatcher(cfg.ExcludeList)
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}
	for _, p := range cfg.WatchList {
		if err := w.Add(p); err != nil {
			return nil, fmt.Errorf("add watch path %q: %w", p, err)
		}
	}

	collector := svc.NewCollector(cfg.StatsDir)

	scanInterval, err := cfg.ScanIntervalDuration()
	if err != nil {
		return nil, fmt.Errorf("parse scan interval: %w", err)
	}
	batchPeriod, err := cfg.BatchPeriodDuration()
	if err != nil {
		return nil, fmt.Errorf("parse batch period: %w", err)
	}

	onJobErr := func(name string, err error) {
		fmt.Fprintf(os.Stderr, "scheduler job %q error: %v\n", name, err)
	}

	sched := scheduler.New()
	sched.Add(scheduler.Job{
		Name:     "scan",
		Interval: scanInterval,
		Fn: func(ctx context.Context) error {
			entries, err := w.Scan(ctx)
			if err != nil {
				return err
			}
			for _, e := range entries {
				collector.AddEntry(e.Path, e.Size, e.LastModified)
			}
			return nil
		},
		OnError: onJobErr,
	})
	sched.Add(scheduler.Job{
		Name:     "flush",
		Interval: batchPeriod,
		Fn: func(ctx context.Context) error {
			if err := collector.FlushHour(ctx); err != nil {
				return err
			}
			collector.Clear()
			return collector.Prune(ctx, cfg.StatsRetentionDays)
		},
		OnError: onJobErr,
	})

	return &runtime{
		watcher:   w,
		collector: collector,
		sched:     sched,
	}, nil
}

// Run starts the scheduler and blocks until ctx is cancelled. On
// shutdown it performs a final flush and closes the watcher.
func Run(ctx context.Context, r *runtime) error {
	// Scheduler.Start blocks until ctx is cancelled, then returns ctx.Err().
	// Cancellation is the expected exit path, so only escalate other errors.
	if err := r.sched.Start(ctx); err != nil &&
		!errors.Is(err, context.Canceled) &&
		!errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("start scheduler: %w", err)
	}

	// The parent ctx is already cancelled — use a fresh, bounded one
	// so the final flush and notification can actually complete.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	finalFlush(shutdownCtx, r)
	r.watcher.Close()
	return nil
}

// finalFlush drains warnings, flushes and prunes stats, then notifies.
// It owns the shutdown lifecycle that used to live in Scheduler.FlushNow.
func finalFlush(ctx context.Context, r *runtime) {
	// Collect warnings from the watcher.
	var warnings []string
	if r.watcher != nil {
		warnings = r.watcher.GetWarnings()
	}

	if err := r.collector.FlushHour(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "flush error: %v\n", err)
	}
	r.collector.Clear()

	if err := r.collector.Prune(ctx, config.GlobalSettings.StatsRetentionDays); err != nil {
		fmt.Fprintf(os.Stderr, "prune error: %v\n", err)
	}

	message := fmt.Sprintf("[%s] Stats flushed and pruned", time.Now().Format(time.RFC3339))
	if len(warnings) > 0 {
		var b strings.Builder
		b.WriteString("\n\nWarnings during file monitoring:")
		for _, w := range warnings {
			b.WriteString("\n- ")
			b.WriteString(w)
		}
		message += b.String()
	}

	if err := config.NewNotifier().Notify(ctx, message); err != nil {
		fmt.Fprintf(os.Stderr, "notify error: %v\n", err)
	}
}
