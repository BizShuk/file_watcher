package svc

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCollector_NewCollector(t *testing.T) {
	c := NewCollector(t.TempDir())
	if c == nil {
		t.Fatal("expected non-nil collector")
	}
	if c.data == nil {
		t.Error("expected data map to be initialized")
	}
}

func TestCollector_Clear(t *testing.T) {
	c := NewCollector(t.TempDir())
	c.AddEntry("/tmp/test.log", 1024, time.Now())
	c.Clear()

	c.mu.RLock()
	if len(c.data) != 0 {
		t.Errorf("expected empty data after Clear, got %d", len(c.data))
	}
	c.mu.RUnlock()
}

func TestCollector_FlushHour_empty(t *testing.T) {
	c := NewCollector(t.TempDir())
	err := c.FlushHour(context.Background())
	if err != nil {
		t.Fatalf("expected no error for empty flush, got %v", err)
	}
}

func TestCollector_FlushHour(t *testing.T) {
	tmpDir := t.TempDir()
	c := NewCollector(tmpDir)

	c.AddEntry("/tmp/a.log", 1024, time.Now())
	c.AddEntry("/tmp/b.log", 2048, time.Now())

	err := c.FlushHour(context.Background())
	if err != nil {
		t.Fatalf("FlushHour returned error: %v", err)
	}

	filename := c.hour.Format("2006-01-02T15") + ".json"
	fpath := filepath.Join(tmpDir, filename)

	data, err := os.ReadFile(fpath)
	if err != nil {
		t.Fatalf("expected stats file at %s, got error: %v", fpath, err)
	}

	var sf StatFile
	if err := json.Unmarshal(data, &sf); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}

	if len(sf.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(sf.Entries))
	}
	if sf.Date == "" {
		t.Error("expected non-empty date")
	}
}

func TestCollector_FlushHour_and_Clear(t *testing.T) {
	c := NewCollector(t.TempDir())
	c.AddEntry("/tmp/test.log", 1024, time.Now())
	c.FlushHour(context.Background())
	c.Clear()

	c.mu.RLock()
	if len(c.data) != 0 {
		t.Errorf("expected empty data after Clear, got %d", len(c.data))
	}
	c.mu.RUnlock()
}

func TestCollector_RoundHour(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2025-05-17T07:30:00Z", "2025-05-17T07:00:00Z"},
		{"2025-05-17T07:00:00Z", "2025-05-17T07:00:00Z"},
		{"2025-05-17T07:59:59Z", "2025-05-17T07:00:00Z"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tm, _ := time.Parse(time.RFC3339, tt.input)
			got := roundHour(tm).Format(time.RFC3339)
			if got != tt.expected {
				t.Errorf("roundHour(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0B"},
		{100, "100B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{1572864, "1.5MB"},
		{1073741824, "1.0GB"},
		{1610612736, "1.5GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatBytes(tt.input)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestComputeGrowth(t *testing.T) {
	entries := map[string][]Entry{
		"/tmp/test1.txt": {
			{Path: "/tmp/test1.txt", Size: 1000, LastModified: time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)},
			{Path: "/tmp/test1.txt", Size: 2000, LastModified: time.Date(2026, 5, 18, 11, 0, 0, 0, time.UTC)},
		},
		"/tmp/test2.txt": {
			{Path: "/tmp/test2.txt", Size: 5000, LastModified: time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)},
			{Path: "/tmp/test2.txt", Size: 4500, LastModified: time.Date(2026, 5, 18, 11, 0, 0, 0, time.UTC)},
		},
	}

	growth := ComputeGrowth(entries)

	if len(growth) != 2 {
		t.Errorf("expected 2 entries, got %d", len(growth))
	}

	growthMap := make(map[string]GrowthEntry)
	for _, g := range growth {
		growthMap[g.Path] = g
	}

	if e, ok := growthMap["/tmp/test1.txt"]; !ok {
		t.Errorf("expected entry for /tmp/test1.txt")
	} else if e.SizeChange != 1000 {
		t.Errorf("expected SizeChange=1000 for test1.txt, got %d", e.SizeChange)
	}

	if e, ok := growthMap["/tmp/test2.txt"]; !ok {
		t.Errorf("expected entry for /tmp/test2.txt")
	} else if e.SizeChange != -500 {
		t.Errorf("expected SizeChange=-500 for test2.txt, got %d", e.SizeChange)
	}
}

func TestComputeGrowthNewFile(t *testing.T) {
	entries := map[string][]Entry{
		"/tmp/new.txt": {
			{Path: "/tmp/new.txt", Size: 5000, LastModified: time.Date(2026, 5, 18, 11, 0, 0, 0, time.UTC)},
		},
	}

	growth := ComputeGrowth(entries)

	if len(growth) != 1 {
		t.Errorf("expected 1 entry, got %d", len(growth))
	}
	if !growth[0].IsNew {
		t.Errorf("expected IsNew=true, got %v", growth[0].IsNew)
	}
}
