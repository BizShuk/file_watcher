// Package config provides configuration loading for file_watcher.
// Settings are loaded from ~/.config/file_watcher/settings.json with
// auto-creation from embedded defaults if the file does not exist.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shuk/file_watcher/utils"
)

var defaultConfigJSON = `{
  "watch_list": [
    "~/projects"
  ],
  "exclude_list": [
    ".git"
  ],
  "admin": {
    "name": "admin",
    "email": "admin@localhost",
    "webhook_url": ""
  },
  "batch_period": "1h",
  "scan_interval": "30m",
  "stats_retention_days": 7
}`

// Settings holds the entire configuration.
type Settings struct {
	WatchList          []string `json:"watch_list"`
	ExcludeList        []string `json:"exclude_list"`
	Admin              Admin    `json:"admin"`
	BatchPeriod        string   `json:"batch_period"`
	ScanInterval       string   `json:"scan_interval"`
	StatsRetentionDays int      `json:"stats_retention_days"`
}

// Admin is reserved for future admin-notification integration; currently unread.
type Admin struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	WebhookURL string `json:"webhook_url"`
}

// Loader loads configuration from the default path.
type Loader struct {
	homeDir string
}

// NewLoader creates a Loader with the given home directory.
func NewLoader(homeDir string) *Loader {
	return &Loader{homeDir: homeDir}
}

// Load reads and parses the settings JSON file from the default config path.
// If the file does not exist, it creates one from the embedded default.
func (l *Loader) Load() (*Settings, error) {
	path := l.configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return l.loadFromDefault()
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	return l.parse(data)
}

// loadFrom reads and parses a settings JSON file from the given path.
func (l *Loader) loadFrom(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return l.parse(data)
}

// loadFromDefault creates a config file from embedded defaults and loads it.
func (l *Loader) loadFromDefault() (*Settings, error) {
	cfg := &Settings{}
	if err := utils.LoadOrCreate(l.configPath(), defaultConfigJSON, cfg); err != nil {
		return nil, err
	}
	return l.parse([]byte(defaultConfigJSON))
}

// parse validates and normalizes the config data.
func (l *Loader) parse(data []byte) (*Settings, error) {
	var cfg Settings
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.BatchPeriod == "" {
		cfg.BatchPeriod = "1h"
	}
	if _, err := time.ParseDuration(cfg.BatchPeriod); err != nil {
		return nil, fmt.Errorf("batch_period %q is not a valid duration: %w", cfg.BatchPeriod, err)
	}
	if cfg.StatsRetentionDays <= 0 {
		cfg.StatsRetentionDays = 7
	}

	cfg.ExpandPaths(l.homeDir)

	if cfg.ScanInterval == "" {
		cfg.ScanInterval = "30m"
	}
	if _, err := time.ParseDuration(cfg.ScanInterval); err != nil {
		return nil, fmt.Errorf("scan_interval %q is not a valid duration: %w", cfg.ScanInterval, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// configPath returns the config file path: ~/.config/file_watcher/settings.json
func (l *Loader) configPath() string {
	return filepath.Join(l.homeDir, ".config", "file_watcher", "settings.json")
}

// BatchPeriodDuration returns the parsed batch period as time.Duration.
func (s *Settings) BatchPeriodDuration() (time.Duration, error) {
	return time.ParseDuration(s.BatchPeriod)
}

// Validate checks if the settings are valid.
func (s *Settings) Validate() error {
	if s.WatchList == nil {
		return fmt.Errorf("missing watch_list")
	}
	if len(s.WatchList) == 0 {
		return fmt.Errorf("empty watch_list")
	}
	for _, p := range s.WatchList {
		if p == "" {
			return fmt.Errorf("empty path in watch_list")
		}
	}
	return nil
}

// ExpandPaths expands tilde (~) characters in path configurations.
func (s *Settings) ExpandPaths(homeDir string) {
	for i, p := range s.WatchList {
		s.WatchList[i] = expandTilde(p, homeDir)
	}
}

func expandTilde(path string, homeDir string) string {
	if path == "~" {
		return homeDir
	}
	if len(path) > 2 && path[:2] == "~/" {
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// ScanIntervalDuration returns the parsed scan interval as time.Duration.
func (s *Settings) ScanIntervalDuration() (time.Duration, error) {
	if s.ScanInterval == "" {
		s.ScanInterval = "30m"
	}
	return time.ParseDuration(s.ScanInterval)
}