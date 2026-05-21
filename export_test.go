package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/shuk/file_watcher/config"
)

func TestRunExport(t *testing.T) {
	tmpDir := t.TempDir()
	cfgLoader := config.NewLoader(tmpDir)
	_, err := cfgLoader.Load()
	if err != nil {
		t.Fatalf("failed to load default config: %v", err)
	}

	var buf bytes.Buffer
	err = runExport(&buf)
	if err != nil {
		t.Fatalf("runExport failed: %v", err)
	}

	var output config.Settings
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("cannot parse output JSON: %v, output: %s", err, buf.String())
	}

	if len(output.WatchList) == 0 {
		t.Errorf("WatchList should not be empty")
	}
	if output.StatsRetentionDays != 7 {
		t.Errorf("expected StatsRetentionDays=7, got %d", output.StatsRetentionDays)
	}
}