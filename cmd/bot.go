/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/amirdaaee/TGMon/internal/app"
	"github.com/amirdaaee/TGMon/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// botCmd represents the bot command
var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "Start TGmon bot",
	Run: func(cmd *cobra.Command, args []string) {
		ll := logrus.WithField("at", "bot")
		ll.Info("starting bot")
		// ...
		cfg := config.Config()
		myBot, err := app.InitializeBot(cfg)
		if err != nil {
			logrus.WithError(err).Fatal("can not build bot")
		}
		ll.Info("bot built")
		// ...
		ll.Warn("starting listening for messages")
		if err := myBot.Start(); err != nil {
			logrus.WithError(err).Fatal("can not start bot")
		}
	},
}

func init() {
	rootCmd.AddCommand(botCmd)
}
