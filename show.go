package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ShowCmd runs the show subcommand to display disk usage growth.
func ShowCmd() error {
	statsDir := defaultStatsDir()

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

	printBarChart(growth)
	return nil
}

// GrowthEntry holds the computed growth for a file path.
type GrowthEntry struct {
	Path         string
	InitialSize  int64
	LatestSize   int64
	SizeChange   int64
	GrowthPct    float64
	IsNew        bool
}

// readAllStats reads all stat files and returns a map of path -> sorted entries by time.
func readAllStats(statsDir string) (map[string][]StatFileEntry, error) {
	result := make(map[string][]StatFileEntry)

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
func computeGrowth(entries map[string][]StatFileEntry) []GrowthEntry {
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

// formatBytes converts bytes to human-readable string.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f%s", float64(bytes)/float64(div), units[exp])
}

// printBarChart renders the growth entries as a horizontal bar chart.
func printBarChart(entries []GrowthEntry) {
	fmt.Println("磁碟使用量增長報告（初始 vs 最新）")
	fmt.Println("================================================================================")
	fmt.Println()

	const maxWidth = 60
	var maxChange int64
	for _, e := range entries {
		if e.SizeChange > maxChange {
			maxChange = e.SizeChange
		}
	}

	for _, e := range entries {
		var barLen int
		if maxChange > 0 {
			barLen = int(float64(e.SizeChange) / float64(maxChange) * float64(maxWidth))
		}
		bar := strings.Repeat("█", barLen)

		sizeStr := formatBytes(e.SizeChange)
		if e.IsNew {
			sizeStr = formatBytes(e.LatestSize) + " (NEW)"
		} else if e.SizeChange == 0 {
			sizeStr = "- (0%)"
		} else {
			sizeStr = fmt.Sprintf("%s (+%.0f%%)", sizeStr, e.GrowthPct)
		}

		// Truncate path if too long
		path := e.Path
		if len(path) > 50 {
			path = "..." + path[len(path)-47:]
		}

		fmt.Printf("%-50s %s  %s\n", path, bar, sizeStr)
	}

	fmt.Println()
	fmt.Println("Legend: bar = growth amount, parentheses = growth percentage")
}

// parseTime parses an RFC3339 time string for testing.
func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}