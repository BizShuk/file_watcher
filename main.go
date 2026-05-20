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

// runShow executes the show subcommand.
func runShow() error {
	return ShowCmd()
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "show":
			if err := runShow(); err != nil {
				fmt.Fprintf(os.Stderr, "show: %v\n", err)
				os.Exit(1)
			}
			return
		case "export":
			if err := runExport(os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "export: %v\n", err)
				os.Exit(1)
			}
			return
		case "test-slack":
			if err := runTestSlack(); err != nil {
				fmt.Fprintf(os.Stderr, "test-slack: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

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
	for _, warn := range watcher.GetWarnings() {
		collector.AddWarning(warn)
	}
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

	go func() {
		if err := watcher.Start(handler); err != nil {
			fmt.Fprintf(os.Stderr, "start watcher: %v\n", err)
			os.Exit(1)
		}
	}()

	var notifiers []Notifier
	notifiers = append(notifiers, &StdoutNotifier{})

	slackToken := os.Getenv("SLACK_BOT_TOKEN")
	slackChannel := os.Getenv("SLACK_CHANNEL_ID")
	fmt.Println("Slack 憑證已偵測 (Slack credentials detected)", "token", slackToken, "channel", slackChannel)
	if slackToken != "" && slackChannel != "" {
		log.Info("Slack 憑證已偵測 (Slack credentials detected)", "channel", slackChannel)
		slackNotifier := NewSlackNotifier(slackToken, slackChannel)
		notifiers = append(notifiers, slackNotifier)

		// 於啟動時發送測試訊息 (Send test message at startup)
		testMsg := fmt.Sprintf("Slack 啟動測試訊息 (Slack Startup Test Message) - 啟動時間: %s", time.Now().Format(time.RFC3339))
		log.Info("正在發送啟動測試訊息至 Slack (Sending startup test message to Slack)...")
		if err := slackNotifier.Notify(testMsg); err != nil {
			log.Error("發送啟動測試訊息至 Slack 失敗 (Failed to send startup test message to Slack)", "error", err)
		} else {
			log.Info("啟動測試訊息發送成功 (Startup test message sent successfully)")
		}
	} else {
		log.Info("未偵測到 Slack 憑證 (Slack credentials not detected)")
	}

	notifier := NewMultiNotifier(notifiers...)
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
