// Package config provides configuration loading for file_watcher.
// Settings are loaded from the specified config directory
// via gosdk config + viper.
package config

import (
	_ "embed"
	"errors"
	"path/filepath"
	"time"

	"github.com/bizshuk/gosdk/config"
	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
)

//go:embed default_settings.json
var defaultSettingsJSON string

// Settings holds the entire configuration.
type Settings struct {
	WatchList          []string `json:"watch_list"`
	ExcludeList        []string `json:"exclude_list"`
	Admin              Admin    `json:"admin"`
	BatchPeriod        string   `json:"batch_period"`
	ScanInterval       string   `json:"scan_interval"`
	StatsRetentionDays int      `json:"stats_retention_days"`
	StatsDir           string   `json:"stats_dir"`
}

// Admin is reserved for future admin-notification integration;
// currently unread.
type Admin struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	WebhookURL string `json:"webhook_url"`
}

// GlobalSettings holds the loaded configuration.
var GlobalSettings *Settings

// Default reads configuration from ~/.config/file_watcher/settings.json
// using gosdk config.DefaultWithDir and viper unmarshal.
func Default() error {
	configDir := config.ExpandHome("~/.config/file_watcher")
	config.Default(
		config.WithAppName("file_watcher"),
		config.WithConfigDir(configDir),
		config.WithDefaultValue(defaultSettingsJSON),
	)

	err := viper.Unmarshal(&GlobalSettings)
	if err != nil {
		return err
	}

	if err := GlobalSettings.validate(); err != nil {
		return err
	}

	log.Info("Loaded settings",
		"watch_list", GlobalSettings.WatchList,
		"exclude_list", GlobalSettings.ExcludeList,
		"batch_period", GlobalSettings.BatchPeriod,
		"scan_interval", GlobalSettings.ScanInterval,
		"stats_retention_days", GlobalSettings.StatsRetentionDays,
		"stats_dir", GlobalSettings.StatsDir,
	)

	return nil
}

// validate checks if the settings are valid.
func (s *Settings) validate() error {
	if s.StatsDir == "" {
		s.StatsDir = filepath.Join(config.GetAppConfigDir(), "stats")
	}

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
		return errors.New("missing watch_list")
	}
	if len(s.WatchList) == 0 {
		return errors.New("empty watch_list")
	}
	for _, p := range s.WatchList {
		if p == "" {
			return errors.New("empty path in watch_list")
		}
	}

	s.ExpandPaths()
	return nil
}

// ExpandPaths expands tilde (~) characters in path configurations.
func (s *Settings) ExpandPaths() {
	for i, p := range s.WatchList {
		s.WatchList[i] = config.ExpandHome(p)
	}
	if s.StatsDir != "" {
		s.StatsDir = config.ExpandHome(s.StatsDir)
	}
}

// BatchPeriodDuration returns the parsed batch period as time.Duration.
func (s *Settings) BatchPeriodDuration() (time.Duration, error) {
	return time.ParseDuration(s.BatchPeriod)
}

// ScanIntervalDuration returns the parsed scan interval as time.Duration.
func (s *Settings) ScanIntervalDuration() (time.Duration, error) {
	return time.ParseDuration(s.ScanInterval)
}
