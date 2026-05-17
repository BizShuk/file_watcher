package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
)

// ErrConfigNotFound is returned when the config file does not exist
// and no default JSON was provided to create one.
var ErrConfigNotFound = errors.New("config file not found")

// LoadOrCreate reads a JSON config file from path.
// If the file does not exist, it creates one from defaultJSON.
// The parsed content is stored in out (which must be a pointer).
//
// Usage:
//
//	type MyConfig struct { Host string `json:"host"` }
//	var cfg MyConfig
//	err := utils.LoadOrCreate("/path/to/config.json", `{"host":"localhost"}`, &cfg)
func LoadOrCreate(path string, defaultJSON string, out interface{}) error {
	// Ensure parent directory exists
	dir := dir(path)
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create config dir %s: %w", dir, err)
		}
	}

	// Check if config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Info("Writing default config to ", path)
		if err := os.WriteFile(path, []byte(defaultJSON), 0644); err != nil {
			return fmt.Errorf("write default config to %s: %w", path, err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %s: %w", path, err)
	}

	log.Info("Loading config from ", path, string(data))

	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}

	return nil
}

// dir returns the directory component of path (empty if no parent).
func dir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return ""
}
