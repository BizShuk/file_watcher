package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StatsCollector defines operations on file-metric collection.
// (ISP + DIP: consumer depends on this small interface)
type StatsCollector interface {
	AddOrUpdate(path string, size int64, modTime time.Time)
	Remove(path string)
	FlushHour() error
	Prune(retentionDays int) error
	Clear()
}

// StatEntry holds the latest size and modification time for a file path.
type StatEntry struct {
	Size         int64     `json:"size_bytes"`
	LastModified time.Time `json:"last_modified"`
}

// StatFile is the JSON structure written to disk each hour.
type StatFile struct {
	Date    string     `json:"date"`
	Entries []StatFileEntry `json:"entries"`
}

type StatFileEntry struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size_bytes"`
	LastModified time.Time `json:"last_modified"`
}

// fsStatsCollector implements StatsCollector.
// Thread-safe via sync.RWMutex (Go concurrency principle: err on side of caution).
type fsStatsCollector struct {
	mu         sync.RWMutex
	data       map[string]StatEntry
	hour       time.Time // the hour bucket being collected
	statsDirFn func() string // injectable for testing
}

// NewStatsCollector creates a new collector starting at the current hour.
func NewStatsCollector() *fsStatsCollector {
	return &fsStatsCollector{
		data:       make(map[string]StatEntry),
		hour:       roundHour(time.Now()),
		statsDirFn: defaultStatsDir,
	}
}

// roundHour returns t rounded down to the start of its hour.
func roundHour(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
}

// AddOrUpdate records or updates the stat for a file path.
// Safe for concurrent use.
func (c *fsStatsCollector) AddOrUpdate(path string, size int64, modTime time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[path] = StatEntry{Size: size, LastModified: modTime}
}

// Remove deletes the stat for a file path.
// Safe for concurrent use.
func (c *fsStatsCollector) Remove(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, path)
}

// FlushHour writes the current hour's data to disk and clears the map.
// Filename: YYYY-MM-DDTHH.json in statsDir.
// (OFP: new output format only adds new code, existing code unchanged)
func (c *fsStatsCollector) FlushHour() error {
	c.mu.Lock()
	entries := make([]StatFileEntry, 0, len(c.data))
	for path, entry := range c.data {
		entries = append(entries, StatFileEntry{
			Path:         path,
			Size:         entry.Size,
			LastModified: entry.LastModified,
		})
	}
	c.mu.Unlock()

	if len(entries) == 0 {
		return nil // nothing to flush
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
	dir := c.statsDirFn()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create stats dir: %w", err)
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write stats file: %w", err)
	}

	return nil
}

// Clear resets the collector and advances to the new hour bucket.
// Called after each successful FlushHour.
func (c *fsStatsCollector) Clear() {
	c.mu.Lock()
	c.data = make(map[string]StatEntry)
	c.hour = roundHour(time.Now())
	c.mu.Unlock()
}

// Prune deletes stat files older than retentionDays in statsDir.
func (c *fsStatsCollector) Prune(retentionDays int) error {
	dir := c.statsDirFn()
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

// statsDir returns the configured stats directory.
func statsDir() string {
	return defaultStatsDir()
}

// defaultStatsDir is the production implementation.
func defaultStatsDir() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "file_watcher", "stats")
}