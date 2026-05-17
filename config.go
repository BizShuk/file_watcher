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
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
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

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// validate checks required fields and sensible defaults.
func (s *Settings) validate() error {
	if len(s.WatchList) == 0 {
		return fmt.Errorf("watch_list is required and must not be empty")
	}
	for _, p := range s.WatchList {
		if p == "" {
			return fmt.Errorf("watch_list contains empty path")
		}
	}
	if s.BatchPeriod == "" {
		s.BatchPeriod = "1h" // default
	}
	if _, err := time.ParseDuration(s.BatchPeriod); err != nil {
		return fmt.Errorf("batch_period %q is not a valid duration: %w", s.BatchPeriod, err)
	}
	if s.StatsRetentionDays <= 0 {
		s.StatsRetentionDays = 7 // default
	}
	return nil
}

// BatchPeriodDuration returns the parsed batch period as time.Duration.
func (s *Settings) BatchPeriodDuration() (time.Duration, error) {
	return time.ParseDuration(s.BatchPeriod)
}