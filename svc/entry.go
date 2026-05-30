package svc

import "time"

// Entry holds the latest size and modification time for a file path.
// Unified from StatEntry + StatFileEntry.
type Entry struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size_bytes"`
	LastModified time.Time `json:"last_modified"`
}

// StatFile is the JSON structure written to disk each hour.
type StatFile struct {
	Date    string  `json:"date"`
	Entries []Entry `json:"entries"`
}

// GrowthEntry holds the computed growth for a file path.
type GrowthEntry struct {
	Path        string
	InitialSize int64
	LatestSize  int64
	SizeChange  int64
	GrowthPct   float64
	IsNew       bool
}
