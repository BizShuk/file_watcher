package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/shuk/file_watcher/config"
	"github.com/spf13/cobra"
)

var ExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export configuration as JSON",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunExport(cmd.OutOrStdout())
	},
}

// RunExport reads the config file and writes it as formatted JSON to the io.Writer.
func RunExport(w io.Writer) error {
	cfg, err := config.Default()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}

	_, err = fmt.Fprintln(w, string(data))
	if err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

func init() {
	RootCmd.AddCommand(ExportCmd)
}
