package show

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/shuk/file_watcher/stats"
)

// StatFile is the JSON structure written to disk each hour.
type StatFile struct {
	Date    string        `json:"date"`
	Entries []stats.Entry `json:"entries"`
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

// ShowCmd runs the show subcommand to display disk usage growth.
func ShowCmd(statsDir string) error {
	entries, err := readAllStats(statsDir)
	if err != nil {
		return fmt.Errorf("read stats: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("目前沒有任何統計資料")
		return nil
	}

	growth := computeGrowth(entries)
	if len(growth) == 0 {
		fmt.Println("無法計算增長資料")
		return nil
	}

	PrintBarChart(growth)
	return nil
}

// readAllStats reads all stat files and returns a map of path -> sorted entries by time.
func readAllStats(statsDir string) (map[string][]stats.Entry, error) {
	result := make(map[string][]stats.Entry)

	patterns := []string{statsDir + "/*.json"}
	var files []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue // skip unreadable files
		}

		var statFile StatFile
		if err := json.Unmarshal(data, &statFile); err != nil {
			continue
		}

		for _, entry := range statFile.Entries {
			result[entry.Path] = append(result[entry.Path], entry)
		}
	}

	// Sort each path's entries by LastModified time
	for path, entries := range result {
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].LastModified.Before(entries[j].LastModified)
		})
		result[path] = entries
	}

	return result, nil
}

// computeGrowth calculates size change from initial to latest for each path.
func computeGrowth(entries map[string][]stats.Entry) []GrowthEntry {
	var growth []GrowthEntry

	for path, pathEntries := range entries {
		if len(pathEntries) < 1 {
			continue
		}

		initial := pathEntries[0]
		latest := pathEntries[len(pathEntries)-1]

		initialSize := initial.Size
		latestSize := latest.Size
		sizeChange := latestSize - initialSize

		isNew := len(pathEntries) == 1 && initialSize > 0

		var growthPct float64
		if initialSize > 0 {
			growthPct = float64(sizeChange) / float64(initialSize) * 100
		}

		growth = append(growth, GrowthEntry{
			Path:        path,
			InitialSize: initialSize,
			LatestSize:  latestSize,
			SizeChange:  sizeChange,
			GrowthPct:   growthPct,
			IsNew:       isNew,
		})
	}

	// Sort by absolute size change descending
	sort.Slice(growth, func(i, j int) bool {
		return growth[i].SizeChange > growth[j].SizeChange
	})

	// Limit to top 20
	if len(growth) > 20 {
		growth = growth[:20]
	}

	return growth
}