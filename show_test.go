package main

import (
	"testing"
)

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
			result := formatBytes(tt.input)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestComputeGrowth(t *testing.T) {
	entries := map[string][]StatFileEntry{
		"/tmp/test1.txt": {
			{Path: "/tmp/test1.txt", Size: 1000, LastModified: parseTime("2026-05-18T10:00:00Z")},
			{Path: "/tmp/test1.txt", Size: 2000, LastModified: parseTime("2026-05-18T11:00:00Z")},
		},
		"/tmp/test2.txt": {
			{Path: "/tmp/test2.txt", Size: 5000, LastModified: parseTime("2026-05-18T10:00:00Z")},
			{Path: "/tmp/test2.txt", Size: 4500, LastModified: parseTime("2026-05-18T11:00:00Z")},
		},
	}

	growth := computeGrowth(entries)

	if len(growth) != 2 {
		t.Errorf("expected 2 entries, got %d", len(growth))
	}

	// First entry should be test1.txt (1000 growth)
	if growth[0].Path != "/tmp/test1.txt" {
		t.Errorf("expected /tmp/test1.txt, got %s", growth[0].Path)
	}
	if growth[0].SizeChange != 1000 {
		t.Errorf("expected SizeChange=1000, got %d", growth[0].SizeChange)
	}

	// Second entry should be test2.txt (-500 growth)
	if growth[1].Path != "/tmp/test2.txt" {
		t.Errorf("expected /tmp/test2.txt, got %s", growth[1].Path)
	}
	if growth[1].SizeChange != -500 {
		t.Errorf("expected SizeChange=-500, got %d", growth[1].SizeChange)
	}
}

func TestComputeGrowthNewFile(t *testing.T) {
	entries := map[string][]StatFileEntry{
		"/tmp/new.txt": {
			{Path: "/tmp/new.txt", Size: 5000, LastModified: parseTime("2026-05-18T11:00:00Z")},
		},
	}

	growth := computeGrowth(entries)

	if len(growth) != 1 {
		t.Errorf("expected 1 entry, got %d", len(growth))
	}
	if !growth[0].IsNew {
		t.Errorf("expected IsNew=true, got %v", growth[0].IsNew)
	}
}