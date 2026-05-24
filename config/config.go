// Package config provides configuration loading for file_watcher.
// Settings are loaded from the specified config directory
// via gosdk config + viper.
package config

import (
	"path/filepath"
	"time"

	"github.com/bizshuk/gosdk/config"
	sdkutils "github.com/bizshuk/gosdk/utils"
	"github.com/charmbracelet/log"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

const defaultConfigFile = "settings.json"

var defaultConfigJSON = `{
    "watch_list": ["~/projects", "~/.hermes", "~/.claude"],
    "exclude_list": [
        ".git",
        "node_modules",
        ".venv",
        "venv",
        "tmp",
        "daemon.err",
        "daemon.log"
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
	StatsDir           string   `json:"stats_dir"`
}

// Admin is reserved for future admin-notification integration;
// currently unread.
type Admin struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	WebhookURL string `json:"webhook_url"`
}

var globalSettings *Settings

// Get returns the loaded global configuration.
func Get() *Settings {
	return globalSettings
}

// Default reads configuration from ~/.config/file_watcher/settings.json
// using gosdk config.DefaultWithDir and viper unmarshal.
func Default() (*Settings, error) {
	configDir := config.ExpandHome("~/.config/file_watcher")

	// Ensure config file exists, auto-create it with defaults if not present.
	configFilePath := filepath.Join(configDir, defaultConfigFile)
	err := sdkutils.CreateIfNotExist(configFilePath, defaultConfigJSON)
	if err != nil {
		return nil, err
	}

	// Use gosdk config.DefaultWithDir to set CONFIG_DIR
	config.DefaultWithDir(configDir)

	var settings Settings
	err = viper.Unmarshal(&settings, func(c *mapstructure.DecoderConfig) {
		c.TagName = "json"
	})
	if err != nil {
		return nil, err
	}

	if settings.StatsDir == "" {
		settings.StatsDir = filepath.Join(configDir, "stats")
	}

	if err := settings.validate(); err != nil {
		return nil, err
	}

	globalSettings = &settings

	log.Info("Loaded settings",
		"watch_list", settings.WatchList,
		"exclude_list", settings.ExcludeList,
		"batch_period", settings.BatchPeriod,
		"scan_interval", settings.ScanInterval,
		"stats_retention_days", settings.StatsRetentionDays,
		"stats_dir", settings.StatsDir,
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

	s.ExpandPaths()
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
