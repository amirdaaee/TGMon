/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// botCmd represents the bot command
var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "Start TGmon bot",
	Run: func(cmd *cobra.Command, args []string) {
		setupLogger()
		ll := logrus.WithField("at", "bot")
		ll.Info("starting bot")
		// ...
		dbContainer, err := buildDbContainer()
		if err != nil {
			logrus.WithError(err).Fatal("can not build db container")
		}
		ll.Info("db container built")
		// ...
		tgClient, err := buildTgClient()
		if err != nil {
			logrus.WithError(err).Fatal("can not build tg client")
		}
		ll.Info("tg client built")
		// ...
		wp, err := buildWorkerContainer()
		if err != nil {
			logrus.WithError(err).Fatal("can not build worker pool")
		}
		// ...
		mediafacade := buildMediaFacade(dbContainer, wp)
		ll.Info("media facade built")
		// ...
		myBot, err := bot.NewBot(tgClient)
		if err != nil {
			logrus.WithError(err).Fatal("can not build bot")
		}
		ll.Info("bot built")
		// ...
		hndler, err := bot.NewHandler(mediafacade, config.Config().TelegramConfig.ChannelID, wp)
		if err != nil {
			logrus.WithError(err).Fatal("can not build bot handler")
		}
		ll.Info("handler built")
		hndler.Register(myBot)
		ll.Info("handler registered")
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
