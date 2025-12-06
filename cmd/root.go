/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

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
