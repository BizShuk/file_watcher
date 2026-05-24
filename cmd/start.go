package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"
	"github.com/bizshuk/file_watcher/config"
	"github.com/bizshuk/file_watcher/handler"
	"github.com/spf13/cobra"
)

var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the file watcher",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := config.Default()
		if err != nil {
			log.Fatal("load config", "err", err)
			return err
		}

		ctx, cancel := context.WithCancel(context.Background())
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sig
			cancel()
		}()

		r, err := handler.Wire()
		if err != nil {
			log.Fatal("wire", "err", err)
			return err
		}

		return handler.Run(ctx, r)
	},
}

func init() {
	RootCmd.AddCommand(StartCmd)
}
