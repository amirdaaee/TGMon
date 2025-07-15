/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/amirdaaee/TGMon/internal/config"
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
		jobReqFacade := buildJobReqFacade(dbContainer)
		jobResFacade := buildJobResFacade(dbContainer)
		ll.Info("media facade built")
		// ...
		g := gin.Default()
		streamHandler := web.NewStreamHandler(dbContainer, mediafacade, wp)
		mediaHandler := web.MediaHandler{}
		jobReqHandler := web.JobReqHandler{}
		jobResHandler := web.JobResHandler{}
		hndlrs := web.HandlerContainer{
			MediaHandler:  web.NewCRDApiHandler(&mediaHandler, mediafacade, "media"),
			JobReqHandler: web.NewCRDApiHandler(&jobReqHandler, jobReqFacade, "jobReq"),
			JobResHandler: web.NewCRDApiHandler(&jobResHandler, jobResFacade, "jobRes"),
		}
		web.RegisterRoutes(g, streamHandler, hndlrs, config.Config().ApiToken)
		ll.Warn("starting server")
		g.Run(":8080")
	},
}

func init() {
	rootCmd.AddCommand(webCmd)
}
