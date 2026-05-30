package svc

import (
	"fmt"
	"strings"
)

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

		sizeStr := FormatBytes(e.SizeChange)
		if e.IsNew {
			sizeStr = FormatBytes(e.LatestSize) + " (NEW)"
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
