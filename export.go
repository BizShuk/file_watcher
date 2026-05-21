package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/shuk/file_watcher/config"
)

// runExport reads the config file and writes it as formatted JSON to the io.Writer.
func runExport(w io.Writer) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	cfgLoader := config.NewLoader(homeDir)
	cfg, err := cfgLoader.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}

	_, err = fmt.Fprintln(w, string(data))
	if err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}