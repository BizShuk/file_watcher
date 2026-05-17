package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/fsnotify/fsnotify"
)

func main() {
	cfg, err := Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	period, err := cfg.BatchPeriodDuration()
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse batch period: %v\n", err)
		os.Exit(1)
	}

	watcher, err := NewWatcher(cfg.ExcludeList)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create watcher: %v\n", err)
		os.Exit(1)
	}
	for _, p := range cfg.WatchList {
		log.Info("add path to watcher", "path", p)
		if err := watcher.Add(p); err != nil {
			fmt.Fprintf(os.Stderr, "add watch path %q: %v\n", p, err)
			os.Exit(1)
		}
	}

	collector := NewStatsCollector()
	handler := func(event fsnotify.Event) {
		var path string = event.Name
		var size int64 = 0
		var modTime int64 = time.Now().Unix()
		fileInfo, err := os.Stat(event.Name)
		if err == nil {
			size = fileInfo.Size()
			modTime = fileInfo.ModTime().Unix()
		}

		if event.Has(fsnotify.Remove) {
			collector.Remove(path)
			return
		}
		collector.AddOrUpdate(path, size, time.Unix(modTime, 0))
	}

	if err := watcher.Start(handler); err != nil {
		fmt.Fprintf(os.Stderr, "start watcher: %v\n", err)
		os.Exit(1)
	}

	notifier := &StdoutNotifier{}
	sched := NewScheduler(collector, notifier, period, cfg.StatsRetentionDays)
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
