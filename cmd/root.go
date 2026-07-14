/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/amirdaaee/TGMon/internal/config"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "TGMon",
	Short: "Telegram media manager",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cfg := config.Config()
		log.Setup(cfg.RuntimeConfig.LogLevel)
		ll := log.GetLogger(log.CmdModule)
		// ...
		ctx, cancel := context.WithCancel(cmd.Context())
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-shutdown
			ll.Warn("Shutting down gracefully...")
			cancel()
		}()
		cmd.SetContext(ctx)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
