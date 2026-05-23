package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/charmbracelet/log"
	"github.com/shuk/file_watcher/config"
	"github.com/shuk/file_watcher/show"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "show":
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "show: %v\n", err)
				os.Exit(1)
			}
			statsDir := filepath.Join(homeDir, ".config", "file_watcher", "stats")
			if err := show.ShowCmd(statsDir); err != nil {
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
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("cannot determine home directory", "err", err)
	}

	cfgLoader := config.NewLoader(homeDir)
	cfg, err := cfgLoader.Load()
	if err != nil {
		log.Fatal("load config", "err", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	r, err := wire(homeDir, cfg)
	if err != nil {
		log.Fatal("wire", "err", err)
	}

	if err := run(ctx, r); err != nil {
		log.Fatal("run", "err", err)
	}
}
