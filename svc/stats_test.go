package svc

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
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

func TestCollector_AddOrUpdate(t *testing.T) {
	c := NewCollector(t.TempDir())
	now := time.Now()

	c.AddOrUpdate("/tmp/test.log", 1024, now)

	c.mu.RLock()
	entry, ok := c.data["/tmp/test.log"]
	c.mu.RUnlock()

	if !ok {
		t.Fatal("expected entry for /tmp/test.log")
	}
	if entry.Size != 1024 {
		t.Errorf("expected size=1024, got %d", entry.Size)
	}
	if !entry.LastModified.Equal(now) {
		t.Errorf("expected modTime=%v, got %v", now, entry.LastModified)
	}
}

func TestCollector_AddOrUpdate_overwrites(t *testing.T) {
	c := NewCollector(t.TempDir())
	now := time.Now()

	c.AddOrUpdate("/tmp/test.log", 1024, now)
	c.AddOrUpdate("/tmp/test.log", 2048, now.Add(time.Hour))

	c.mu.RLock()
	entry := c.data["/tmp/test.log"]
	c.mu.RUnlock()

	if entry.Size != 2048 {
		t.Errorf("expected size=2048, got %d", entry.Size)
	}
}

func TestCollector_Remove(t *testing.T) {
	c := NewCollector(t.TempDir())
	c.AddOrUpdate("/tmp/test.log", 1024, time.Now())
	c.Remove("/tmp/test.log")

	c.mu.RLock()
	_, ok := c.data["/tmp/test.log"]
	c.mu.RUnlock()
	if ok {
		t.Error("expected entry to be removed")
	}
}

func TestCollector_Clear(t *testing.T) {
	c := NewCollector(t.TempDir())
	c.AddOrUpdate("/tmp/test.log", 1024, time.Now())
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

	c.AddOrUpdate("/tmp/a.log", 1024, time.Now())
	c.AddOrUpdate("/tmp/b.log", 2048, time.Now())

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

func TestCollector_FlushHour_clearsData(t *testing.T) {
	c := NewCollector(t.TempDir())
	c.AddOrUpdate("/tmp/test.log", 1024, time.Now())
	c.FlushHour(context.Background())
	c.Clear()

	c.mu.RLock()
	if len(c.data) != 0 {
		t.Errorf("expected empty data after FlushHour+Clear, got %d", len(c.data))
	}
	c.mu.RUnlock()
}

func TestCollector_ConcurrentAddOrUpdate(t *testing.T) {
	c := NewCollector(t.TempDir())
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c.AddOrUpdate("/tmp/file", int64(n), time.Now())
		}(i)
	}
	wg.Wait()
	c.mu.RLock()
	_, ok := c.data["/tmp/file"]
	c.mu.RUnlock()
	if !ok {
		t.Error("expected entry to exist after concurrent writes")
	}
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