package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadFrom(t *testing.T) {
	// Helper to create a temp config file and load it via loadFrom.
	tmp := func(content string) string {
		t.Helper()
		f := filepath.Join(t.TempDir(), "settings.json")
		os.WriteFile(f, []byte(content), 0600)
		return f
	}

	t.Run("valid minimal config", func(t *testing.T) {
		cfg, err := loadFrom(tmp(`{"watch_list": ["/tmp"]}`))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(cfg.WatchList) != 1 || cfg.WatchList[0] != "/tmp" {
			t.Errorf("unexpected watch_list: %v", cfg.WatchList)
		}
	})

	t.Run("default batch_period", func(t *testing.T) {
		cfg, err := loadFrom(tmp(`{"watch_list": ["/tmp"]}`))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cfg.BatchPeriod != "1h" {
			t.Errorf("expected default batch_period=1h, got %q", cfg.BatchPeriod)
		}
	})

	t.Run("default stats_retention_days", func(t *testing.T) {
		cfg, err := loadFrom(tmp(`{"watch_list": ["/tmp"]}`))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cfg.StatsRetentionDays != 7 {
			t.Errorf("expected default stats_retention_days=7, got %d", cfg.StatsRetentionDays)
		}
	})

	t.Run("invalid batch_period", func(t *testing.T) {
		_, err := loadFrom(tmp(`{"watch_list": ["/tmp"], "batch_period": "not-a-duration"}`))
		if err == nil {
			t.Fatal("expected error for invalid batch_period")
		}
	})

	t.Run("missing watch_list", func(t *testing.T) {
		_, err := loadFrom(tmp(`{}`))
		if err == nil {
			t.Fatal("expected error for missing watch_list")
		}
	})

	t.Run("empty watch_list", func(t *testing.T) {
		_, err := loadFrom(tmp(`{"watch_list": []}`))
		if err == nil {
			t.Fatal("expected error for empty watch_list")
		}
	})

	t.Run("empty path in watch_list", func(t *testing.T) {
		_, err := loadFrom(tmp(`{"watch_list": ["/tmp", ""]}`))
		if err == nil {
			t.Fatal("expected error for empty path in watch_list")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := loadFrom("/nonexistent/path/settings.json")
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		_, err := loadFrom(tmp(`{invalid json}`))
		if err == nil {
			t.Fatal("expected error for malformed JSON")
		}
	})
}

func TestBatchPeriodDuration(t *testing.T) {
	t.Run("parse valid duration", func(t *testing.T) {
		cfg := &Settings{BatchPeriod: "30m"}
		d, err := cfg.BatchPeriodDuration()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if d != 30*time.Minute {
			t.Errorf("expected 30m, got %v", d)
		}
	})

	t.Run("parse invalid duration", func(t *testing.T) {
		cfg := &Settings{BatchPeriod: "abc"}
		_, err := cfg.BatchPeriodDuration()
		if err == nil {
			t.Fatal("expected error for invalid duration")
		}
	})
}

func TestDefaultConfigCreation(t *testing.T) {
	tmpDir := t.TempDir()
	origHomeDirFn := homeDirFn
	homeDirFn = func() string {
		return tmpDir
	}
	t.Cleanup(func() {
		homeDirFn = origHomeDirFn
	})

	// Load() should create the file from embedded default
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(cfg.WatchList) != 1 || cfg.WatchList[0] != "/tmp" {
		t.Errorf("unexpected watch_list from default: %v", cfg.WatchList)
	}
	if cfg.BatchPeriod != "1h" {
		t.Errorf("expected default batch_period=1h, got %q", cfg.BatchPeriod)
	}
	if cfg.StatsRetentionDays != 7 {
		t.Errorf("expected default stats_retention_days=7, got %d", cfg.StatsRetentionDays)
	}
}
