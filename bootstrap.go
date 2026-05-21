package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
	"github.com/fsnotify/fsnotify"
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

	// Build watcher
	w, err := watcher.New(cfg.ExcludeList)
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}
	for _, p := range cfg.WatchList {
		if err := w.Add(p); err != nil {
			return nil, fmt.Errorf("add watch path %q: %w", p, err)
		}
	}

	// Build stats collector
	collector := stats.NewCollector(statsDir)

	// Build warning sink
	warnings := warning.NewSink()

	// Build notifier
	notif := buildNotifier()

	// Build scheduler
	period, err := cfg.BatchPeriodDuration()
	if err != nil {
		return nil, fmt.Errorf("parse batch period: %w", err)
	}
	sched := scheduler.New(collector, warnings, notif, period, cfg.StatsRetentionDays)

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
	handler := func(path string, op fsnotify.Op) error {
		var size int64 = 0
		var modTime time.Time = time.Now()
		fileInfo, err := os.Stat(path)
		if err == nil {
			size = fileInfo.Size()
			modTime = fileInfo.ModTime()
		}

		if op.Has(fsnotify.Remove) {
			r.collector.Remove(path)
			return nil
		}
		r.collector.AddOrUpdate(path, size, modTime)
		return nil
	}

	go func() {
		if err := r.watcher.Start(ctx, handler); err != nil {
			log.Error("start watcher", "err", err)
		}
	}()

	if err := r.sched.Start(ctx); err != nil {
		return fmt.Errorf("start scheduler: %w", err)
	}

	<-ctx.Done()
	r.sched.FlushNow()
	r.watcher.Close()
	return nil
}
