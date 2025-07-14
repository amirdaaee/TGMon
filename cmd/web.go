/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/amirdaaee/TGMon/internal/web"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// webCmd represents the web command
var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start TGmon web server",
	Run: func(cmd *cobra.Command, args []string) {
		setupLogger()
		ll := logrus.WithField("at", "web")
		ll.Info("starting web server")
		//...
		dbContainer, err := buildDbContainer()
		if err != nil {
			logrus.WithError(err).Fatal("can not build db container")
		}
		ll.Info("db container built")
		// ...
		wp, err := buildWorkerContainer()
		if err != nil {
			logrus.WithError(err).Fatal("can not build worker pool")
		}
		// ...
		mediafacade := buildMediaFacade(dbContainer, wp)
		ll.Info("media facade built")
		// ...
		g := gin.Default()
		streamHandler := web.NewStreamHandler(dbContainer, mediafacade, wp)
		web.RegisterRoutes(g, streamHandler)
		ll.Warn("starting server")
		g.Run(":8080")
	},
}

func init() {
	rootCmd.AddCommand(webCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// webCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// webCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
