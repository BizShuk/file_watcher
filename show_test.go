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

	// Build a map for easy lookup
	growthMap := make(map[string]GrowthEntry)
	for _, g := range growth {
		growthMap[g.Path] = g
	}

	// Check test1.txt
	if e, ok := growthMap["/tmp/test1.txt"]; !ok {
		t.Errorf("expected entry for /tmp/test1.txt")
	} else if e.SizeChange != 1000 {
		t.Errorf("expected SizeChange=1000 for test1.txt, got %d", e.SizeChange)
	}

	// Check test2.txt
	if e, ok := growthMap["/tmp/test2.txt"]; !ok {
		t.Errorf("expected entry for /tmp/test2.txt")
	} else if e.SizeChange != -500 {
		t.Errorf("expected SizeChange=-500 for test2.txt, got %d", e.SizeChange)
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