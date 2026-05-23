package cmd

import (
	"os"
	"path/filepath"

	"github.com/shuk/file_watcher/show"
	"github.com/spf13/cobra"
)

var ShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show disk usage growth chart",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		statsDir := filepath.Join(homeDir, ".config", "file_watcher", "stats")
		return show.ShowCmd(statsDir)
	},
}

func init() {
	RootCmd.AddCommand(ShowCmd)
	RootCmd.RunE = ShowCmd.RunE
}
