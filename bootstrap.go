package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shuk/file_watcher/config"
	"github.com/shuk/file_watcher/notify"
	"github.com/shuk/file_watcher/scheduler"
	"github.com/shuk/file_watcher/stats"
	"github.com/shuk/file_watcher/watcher"
	"github.com/shuk/file_watcher/warning"
)

// SchedulerOps defines scheduler operations needed by runtime.
type SchedulerOps interface {
	Start(ctx context.Context) error
	FlushNow()
}

// runtime holds the application components started together.
type runtime struct {
	watcher   watcher.Watcher
	collector *stats.Collector
	notifier  notify.Notifier
	sched     SchedulerOps
}

// wire builds the runtime from configuration. It is the sole DI entry point.
func wire(homeDir string, cfg *config.Settings) (*runtime, error) {
	statsDir := filepath.Join(homeDir, ".config", "file_watcher", "stats")

	// Build watcher (FsWatcher)
	w, err := watcher.New(cfg.ExcludeList)
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}
	for _, p := range cfg.WatchList {
		if err := w.Add(p); err != nil {
			return nil, fmt.Errorf("add watch path %q: %w", p, err)
		}
	}

	// Build collector
	collector := stats.NewCollector(statsDir)

	// Build warning sink
	warnings := warning.NewSink()

	// Build notifier
	notif := buildNotifier()

	// Parse intervals
	scanInterval, err := cfg.ScanIntervalDuration()
	if err != nil {
		return nil, fmt.Errorf("parse scan interval: %w", err)
	}
	batchPeriod, err := cfg.BatchPeriodDuration()
	if err != nil {
		return nil, fmt.Errorf("parse batch period: %w", err)
	}

	// Build scheduler with chainable Every() API
	sched := scheduler.New(collector, warnings, notif)

	// Scan job: walk all watch_list paths and update collector
	sched.Every("scan", scanInterval, func(ctx context.Context) error {
		return w.Scan(ctx)
	})

	// Flush job: write stats to disk and prune old files
	sched.Every("flush", batchPeriod, func(ctx context.Context) error {
		if err := collector.FlushHour(ctx); err != nil {
			return err
		}
		collector.Clear()
		return collector.Prune(ctx, cfg.StatsRetentionDays)
	})

	return &runtime{
		watcher:   w,
		collector: collector,
		notifier:  notif,
		sched:     sched,
	}, nil
}

// buildNotifier creates the notifier chain from environment.
func buildNotifier() notify.Notifier {
	var notifiers []notify.Notifier
	notifiers = append(notifiers, &notify.StdoutNotifier{})

	slackToken := os.Getenv("SLACK_BOT_TOKEN")
	slackChannel := os.Getenv("SLACK_CHANNEL_ID")
	if slackToken != "" && slackChannel != "" {
		notifiers = append(notifiers, notify.NewSlackNotifier(slackToken, slackChannel))
	}

	return notify.NewMulti(notifiers...)
}

// run starts the watcher, scheduler, and blocks until signal.
func run(ctx context.Context, r *runtime) error {
	// No more fsnotify goroutine — scan is now scheduler-driven

	if err := r.sched.Start(ctx); err != nil {
		return fmt.Errorf("start scheduler: %w", err)
	}

	<-ctx.Done()
	r.sched.FlushNow()
	r.watcher.Close()
	return nil
}
