/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/amirdaaee/TGMon/cmd/wire"
	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// botCmd represents the bot command
var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "Start TGmon bot",
	Run: func(cmd *cobra.Command, args []string) {
		ll := log.Named(log.CmdModule, "bot")
		// ...
		cntr := wire.GetProvider()

		if err := cntr.Invoke(func(myBot *bot.Bot) error {
			ll.Info("starting bot")
			if err := myBot.Start(cmd.Context()); err != nil {
				return err
			}
			ll.Info("bot stopped")
			return nil
		}); err != nil {
			ll.With(zap.Error(err)).Fatal("can not start bot")
		}
	},
}

func init() {
	rootCmd.AddCommand(botCmd)
}
