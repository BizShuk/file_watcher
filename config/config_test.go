package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSettings_Validate(t *testing.T) {
	t.Run("default batch_period", func(t *testing.T) {
		cfg := &Settings{WatchList: []string{"/tmp"}}
		err := cfg.validate()
		if err != nil {
			t.Fatalf("validate() failed: %v", err)
		}
		if cfg.BatchPeriod != "1h" {
			t.Errorf("expected default batch_period=1h, got %q", cfg.BatchPeriod)
		}
	})

	t.Run("default stats_retention_days", func(t *testing.T) {
		cfg := &Settings{WatchList: []string{"/tmp"}}
		err := cfg.validate()
		if err != nil {
			t.Fatalf("validate() failed: %v", err)
		}
		if cfg.StatsRetentionDays != 7 {
			t.Errorf("expected default stats_retention_days=7, got %d", cfg.StatsRetentionDays)
		}
	})

	t.Run("invalid batch_period", func(t *testing.T) {
		cfg := &Settings{WatchList: []string{"/tmp"}, BatchPeriod: "not-a-duration"}
		err := cfg.validate()
		if err == nil {
			t.Fatal("expected error for invalid batch_period")
		}
	})

	t.Run("missing watch_list", func(t *testing.T) {
		cfg := &Settings{}
		err := cfg.validate()
		if err == nil {
			t.Fatal("expected error for missing watch_list")
		}
	})

	t.Run("empty watch_list", func(t *testing.T) {
		cfg := &Settings{WatchList: []string{}}
		err := cfg.validate()
		if err == nil {
			t.Fatal("expected error for empty watch_list")
		}
	})

	t.Run("empty path in watch_list", func(t *testing.T) {
		cfg := &Settings{WatchList: []string{"/tmp", ""}}
		err := cfg.validate()
		if err == nil {
			t.Fatal("expected error for empty path in watch_list")
		}
	})

	t.Run("expand tilde in watch_list", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)
		cfg := &Settings{WatchList: []string{"~", "~/projects", "/tmp"}}
		cfg.ExpandPaths()

		expected := []string{tmpDir, filepath.Join(tmpDir, "projects"), "/tmp"}
		if len(cfg.WatchList) != len(expected) {
			t.Fatalf("expected %d paths, got %d", len(expected), len(cfg.WatchList))
		}
		for i, p := range cfg.WatchList {
			if p != expected[i] {
				t.Errorf("expected path %q, got %q", expected[i], p)
			}
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

func TestGlobalConfig(t *testing.T) {
	t.Run("set and access global config", func(t *testing.T) {
		cfg := &Settings{WatchList: []string{"/tmp"}}
		GlobalSettings = cfg
		if GlobalSettings != cfg {
			t.Errorf("expected global config %v, got %v", cfg, GlobalSettings)
		}
	})
}
