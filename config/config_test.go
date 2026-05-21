package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoader_LoadFrom(t *testing.T) {
	tmp := func(content string) string {
		t.Helper()
		f := filepath.Join(t.TempDir(), "settings.json")
		os.WriteFile(f, []byte(content), 0600)
		return f
	}

	t.Run("valid minimal config", func(t *testing.T) {
		l := NewLoader(t.TempDir())
		path := tmp(`{"watch_list": ["/tmp"]}`)
		cfg, err := l.loadFrom(path)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(cfg.WatchList) != 1 || cfg.WatchList[0] != "/tmp" {
			t.Errorf("unexpected watch_list: %v", cfg.WatchList)
		}
	})

	t.Run("default batch_period", func(t *testing.T) {
		l := NewLoader(t.TempDir())
		cfg, err := l.loadFrom(tmp(`{"watch_list": ["/tmp"]}`))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cfg.BatchPeriod != "1h" {
			t.Errorf("expected default batch_period=1h, got %q", cfg.BatchPeriod)
		}
	})

	t.Run("default stats_retention_days", func(t *testing.T) {
		l := NewLoader(t.TempDir())
		cfg, err := l.loadFrom(tmp(`{"watch_list": ["/tmp"]}`))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cfg.StatsRetentionDays != 7 {
			t.Errorf("expected default stats_retention_days=7, got %d", cfg.StatsRetentionDays)
		}
	})

	t.Run("invalid batch_period", func(t *testing.T) {
		l := NewLoader(t.TempDir())
		_, err := l.loadFrom(tmp(`{"watch_list": ["/tmp"], "batch_period": "not-a-duration"}`))
		if err == nil {
			t.Fatal("expected error for invalid batch_period")
		}
	})

	t.Run("missing watch_list", func(t *testing.T) {
		l := NewLoader(t.TempDir())
		_, err := l.loadFrom(tmp(`{}`))
		if err == nil {
			t.Fatal("expected error for missing watch_list")
		}
	})

	t.Run("empty watch_list", func(t *testing.T) {
		l := NewLoader(t.TempDir())
		_, err := l.loadFrom(tmp(`{"watch_list": []}`))
		if err == nil {
			t.Fatal("expected error for empty watch_list")
		}
	})

	t.Run("empty path in watch_list", func(t *testing.T) {
		l := NewLoader(t.TempDir())
		_, err := l.loadFrom(tmp(`{"watch_list": ["/tmp", ""]}`))
		if err == nil {
			t.Fatal("expected error for empty path in watch_list")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		l := NewLoader(t.TempDir())
		_, err := l.loadFrom("/nonexistent/path/settings.json")
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		l := NewLoader(t.TempDir())
		_, err := l.loadFrom(tmp(`{invalid json}`))
		if err == nil {
			t.Fatal("expected error for malformed JSON")
		}
	})

	t.Run("expand tilde in watch_list", func(t *testing.T) {
		tmpDir := t.TempDir()
		l := NewLoader(tmpDir)
		cfg, err := l.loadFrom(tmp(`{"watch_list": ["~", "~/projects", "/tmp"]}`))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
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

func TestLoader_Load(t *testing.T) {
	t.Run("load creates default config", func(t *testing.T) {
		tmpDir := t.TempDir()
		l := NewLoader(tmpDir)
		cfg, err := l.Load()
		if err != nil {
			t.Fatalf("Load() failed: %v", err)
		}

		expectedPath := filepath.Join(tmpDir, "projects")
		if len(cfg.WatchList) != 1 || cfg.WatchList[0] != expectedPath {
			t.Errorf("unexpected watch_list from default: %v, expected %q", cfg.WatchList, expectedPath)
		}
		if cfg.BatchPeriod != "1h" {
			t.Errorf("expected default batch_period=1h, got %q", cfg.BatchPeriod)
		}
		if cfg.StatsRetentionDays != 7 {
			t.Errorf("expected default stats_retention_days=7, got %d", cfg.StatsRetentionDays)
		}
	})
}
