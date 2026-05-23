// Package config provides configuration loading for file_watcher.
// Settings are loaded from the specified config directory via gosdk config + viper.
package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/bizshuk/gosdk/config"
	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
)

const defaultConfigFile = "settings.json"

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
	WatchList          []string `json:"watch_list" mapstructure:"watch_list"`
	ExcludeList        []string `json:"exclude_list" mapstructure:"exclude_list"`
	Admin              Admin    `json:"admin" mapstructure:"admin"`
	BatchPeriod        string   `json:"batch_period" mapstructure:"batch_period"`
	ScanInterval       string   `json:"scan_interval" mapstructure:"scan_interval"`
	StatsRetentionDays int      `json:"stats_retention_days" mapstructure:"stats_retention_days"`
}

// Admin is reserved for future admin-notification integration; currently unread.
type Admin struct {
	Name       string `json:"name" mapstructure:"name"`
	Email      string `json:"email" mapstructure:"email"`
	WebhookURL string `json:"webhook_url" mapstructure:"webhook_url"`
}

// Default reads configuration from ~/.config/file_watcher/settings.json
// using gosdk config.DefaultWithDir and viper unmarshal.
func Default() (*Settings, error) {
	homeDir, _ := os.UserHomeDir()
	configDir := expandTilde("~/.config/file_watcher", homeDir)

	// Use gosdk config.DefaultWithDir to set CONFIG_DIR
	config.DefaultWithDir(configDir)

	var settings Settings
	if err := viper.Unmarshal(&settings); err != nil {
		return nil, err
	}

	if err := settings.validate(); err != nil {
		return nil, err
	}

	log.Info("Loaded settings",
		"watch_list", settings.WatchList,
		"exclude_list", settings.ExcludeList,
		"batch_period", settings.BatchPeriod,
		"scan_interval", settings.ScanInterval,
		"stats_retention_days", settings.StatsRetentionDays,
	)

	return &settings, nil
}

// validate checks if the settings are valid.
func (s *Settings) validate() error {
	if s.BatchPeriod == "" {
		s.BatchPeriod = "1h"
	}
	if _, err := time.ParseDuration(s.BatchPeriod); err != nil {
		return err
	}

	if s.StatsRetentionDays <= 0 {
		s.StatsRetentionDays = 7
	}

	if s.ScanInterval == "" {
		s.ScanInterval = "30m"
	}
	if _, err := time.ParseDuration(s.ScanInterval); err != nil {
		return err
	}

	if s.WatchList == nil {
		return errMissingWatchList
	}
	if len(s.WatchList) == 0 {
		return errEmptyWatchList
	}
	for _, p := range s.WatchList {
		if p == "" {
			return errEmptyPathInWatchList
		}
	}

	s.ExpandPaths("")
	return nil
}

var (
	errMissingWatchList     = &configError{"missing watch_list"}
	errEmptyWatchList       = &configError{"empty watch_list"}
	errEmptyPathInWatchList = &configError{"empty path in watch_list"}
)

type configError struct {
	msg string
}

func (e *configError) Error() string { return e.msg }

// ExpandPaths expands tilde (~) characters in path configurations.
func (s *Settings) ExpandPaths(homeDir string) {
	if homeDir == "" {
		homeDir, _ = os.UserHomeDir()
	}
	for i, p := range s.WatchList {
		s.WatchList[i] = expandTilde(p, homeDir)
	}
}

func expandTilde(path string, homeDir string) string {
	if homeDir == "" {
		homeDir, _ = os.UserHomeDir()
	}
	if path == "~" {
		return homeDir
	}
	if len(path) > 2 && path[:2] == "~/" {
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// BatchPeriodDuration returns the parsed batch period as time.Duration.
func (s *Settings) BatchPeriodDuration() (time.Duration, error) {
	return time.ParseDuration(s.BatchPeriod)
}

// ScanIntervalDuration returns the parsed scan interval as time.Duration.
func (s *Settings) ScanIntervalDuration() (time.Duration, error) {
	return time.ParseDuration(s.ScanInterval)
}
