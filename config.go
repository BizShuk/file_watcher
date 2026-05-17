package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/shuk/file_watcher/utils"
)

//go:embed settings.default.json
var defaultConfigJSON string

var homeDirFn = func() string {
	return os.Getenv("HOME")
}

// configPath returns the config file path: ~/.config/file_watcher/settings.json
func configPath() string {
	return configDir() + "/settings.json"
}

// configDir returns ~/.config/file_watcher
func configDir() string {
	return homeDirFn() + "/.config/file_watcher"
}

// Settings holds the entire configuration.
type Settings struct {
	WatchList          []string `json:"watch_list"`
	ExcludeList        []string `json:"exclude_list"`
	Admin              Admin    `json:"admin"`
	BatchPeriod        string   `json:"batch_period"`
	StatsRetentionDays int      `json:"stats_retention_days"`
}

// Admin holds system admin info.
type Admin struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	WebhookURL string `json:"webhook_url"`
}

// Load reads and parses the settings JSON file from the default config path.
// If the file does not exist, it creates one from the embedded default.
func Load() (*Settings, error) {
	var cfg Settings
	if err := utils.LoadOrCreate(configPath(), defaultConfigJSON, &cfg); err != nil {
		return nil, err
	}
	if cfg.BatchPeriod == "" {
		cfg.BatchPeriod = "1h" // default
	}
	if _, err := time.ParseDuration(cfg.BatchPeriod); err != nil {
		return nil, fmt.Errorf("batch_period %q is not a valid duration: %w", cfg.BatchPeriod, err)
	}
	if cfg.StatsRetentionDays <= 0 {
		cfg.StatsRetentionDays = 7 // default
	}

	return &cfg, nil
}

// loadFrom reads and parses an existing settings JSON file (no auto-create).
// Used by tests that provide an explicit path.
func loadFrom(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Settings
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.BatchPeriod == "" {
		cfg.BatchPeriod = "1h" // default
	}
	if _, err := time.ParseDuration(cfg.BatchPeriod); err != nil {
		return nil, fmt.Errorf("batch_period %q is not a valid duration: %w", cfg.BatchPeriod, err)
	}
	if cfg.StatsRetentionDays <= 0 {
		cfg.StatsRetentionDays = 7 // default
	}

	return &cfg, nil
}


// BatchPeriodDuration returns the parsed batch period as time.Duration.
func (s *Settings) BatchPeriodDuration() (time.Duration, error) {
	return time.ParseDuration(s.BatchPeriod)
}