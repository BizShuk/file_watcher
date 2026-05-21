package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Recorder captures file change events from the watcher.
type Recorder interface {
	AddOrUpdate(path string, size int64, modTime time.Time)
	Remove(path string)
}

// Flusher persists collected stats and manages retention.
type Flusher interface {
	FlushHour(ctx context.Context) error
	Clear()
	Prune(ctx context.Context, retentionDays int) error
}

// Collector implements both Recorder and Flusher.
type Collector struct {
	mu       sync.RWMutex
	data     map[string]Entry
	hour     time.Time
	statsDir string
}

// NewCollector creates a new collector with the given stats directory.
func NewCollector(statsDir string) *Collector {
	return &Collector{
		data:     make(map[string]Entry),
		hour:     roundHour(time.Now()),
		statsDir: statsDir,
	}
}

// roundHour returns t rounded down to the start of its hour.
func roundHour(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
}

// AddOrUpdate records or updates the stat for a file path.
// Safe for concurrent use.
func (c *Collector) AddOrUpdate(path string, size int64, modTime time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[path] = Entry{Path: path, Size: size, LastModified: modTime}
}

// Remove deletes the stat for a file path.
// Safe for concurrent use.
func (c *Collector) Remove(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, path)
}

// FlushHour writes the current hour's data to disk and clears the map.
// Filename: YYYY-MM-DDTHH.json in statsDir.
func (c *Collector) FlushHour(ctx context.Context) error {
	c.mu.Lock()
	entries := make([]Entry, 0, len(c.data))
	for _, entry := range c.data {
		entries = append(entries, entry)
	}
	c.mu.Unlock()

	if len(entries) == 0 {
		return nil
	}

	statFile := StatFile{
		Date:    c.hour.Format(time.RFC3339),
		Entries: entries,
	}

	data, err := json.MarshalIndent(statFile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal stats: %w", err)
	}

	filename := c.hour.Format("2006-01-02T15") + ".json"
	dir := c.statsDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create stats dir: %w", err)
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write stats file: %w", err)
	}

	return nil
}

// Clear resets the collector and advances to the new hour bucket.
func (c *Collector) Clear() {
	c.mu.Lock()
	c.data = make(map[string]Entry)
	c.hour = roundHour(time.Now())
	c.mu.Unlock()
}

// Prune deletes stat files older than retentionDays in statsDir.
func (c *Collector) Prune(ctx context.Context, retentionDays int) error {
	dir := c.statsDir
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read stats dir: %w", err)
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
	return nil
}