package cmd

import (
	"fmt"
	"strings"

	"github.com/bizshuk/file_watcher/config"
	"github.com/bizshuk/file_watcher/svc"
	"github.com/spf13/cobra"
)

var ShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show disk usage growth chart",
	RunE: func(cmd *cobra.Command, args []string) error {
		return show()
	},
}

func init() {
	RootCmd.AddCommand(ShowCmd)
	RootCmd.RunE = ShowCmd.RunE
}

// show runs the show subcommand to display disk usage growth.
func show() error {
	entries, err := svc.ReadAllStats(config.GlobalSettings.StatsDir)
	if err != nil {
		return fmt.Errorf("read stats: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("目前沒有任何統計資料")
		return nil
	}

	growth := svc.ComputeGrowth(entries)
	if len(growth) == 0 {
		fmt.Println("無法計算增長資料")
		return nil
	}

	printBarChart(growth)
	return nil
}

// printBarChart renders the growth entries as a horizontal bar chart.
func printBarChart(entries []svc.GrowthEntry) {
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

		sizeStr := svc.FormatBytes(e.SizeChange)
		if e.IsNew {
			sizeStr = svc.FormatBytes(e.LatestSize) + " (NEW)"
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