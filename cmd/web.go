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
		infoHandler := web.InfoApiHandler{
			MediaFacade: mediafacade,
		}
		loginHandler := web.LoginApiHandler{
			UserName: config.Config().UserName,
			UserPass: config.Config().UserPass,
			Token:    config.Config().ApiToken,
		}
		hndlrs := web.HandlerContainer{
			MediaHandler:  web.NewCRDApiHandler(&mediaHandler, mediafacade, "media"),
			JobReqHandler: web.NewCRDApiHandler(&jobReqHandler, jobReqFacade, "jobReq"),
			JobResHandler: web.NewCRDApiHandler(&jobResHandler, jobResFacade, "jobRes"),
			InfoHandler:   web.NewApiHandler(&infoHandler, "info"),
			LoginHandler:  web.NewApiHandler(&loginHandler, "login"),
		}
		cfg := config.Config()
		web.RegisterRoutes(g, streamHandler, hndlrs, cfg.ApiToken, cfg.Swagger)
		ll.Warn("starting server")
		if err := g.Run(":8080"); err != nil {
			logrus.WithError(err).Fatal("error running webserver")
		}
	},
}

func init() {
	rootCmd.AddCommand(webCmd)
}
