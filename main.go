package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg, err := Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	collector := NewStatsCollector()

	notifier := &StdoutNotifier{}

	period, err := cfg.BatchPeriodDuration()
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse batch period: %v\n", err)
		os.Exit(1)
	}
	sched := NewScheduler(collector, notifier, period, cfg.StatsRetentionDays)

	watcher, err := NewWatcher()
	if err != nil {
		fmt.Fprintf(os.Stderr, "create watcher: %v\n", err)
		os.Exit(1)
	}
	for _, p := range cfg.WatchList {
		if err := watcher.Add(p); err != nil {
			fmt.Fprintf(os.Stderr, "add watch path %q: %v\n", p, err)
			os.Exit(1)
		}
	}

	handler := func(path string, size int64, modTime int64, isRemove bool) {
		if isRemove {
			collector.Remove(path)
		} else {
			collector.AddOrUpdate(path, size, time.Unix(modTime, 0))
		}
	}

	if err := watcher.Start(handler); err != nil {
		fmt.Fprintf(os.Stderr, "start watcher: %v\n", err)
		os.Exit(1)
	}

	if err := sched.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "start scheduler: %v\n", err)
		os.Exit(1)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	sched.FlushNow()
	watcher.Close()
}