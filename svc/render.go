package svc

import (
	"fmt"
	"strings"
)

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

// PrintBarChart renders the growth entries as a horizontal bar chart.
func PrintBarChart(entries []GrowthEntry) {
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
