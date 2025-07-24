/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/amirdaaee/TGMon/internal/config"
	"github.com/amirdaaee/TGMon/internal/web"
	"github.com/gin-contrib/cors"
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
		hCfg := config.Config().HttpConfig
		g := gin.Default()
		coresCfg := cors.DefaultConfig()
		if len(hCfg.CoresAllowed) > 0 {
			coresCfg.AllowOrigins = hCfg.CoresAllowed
		} else {
			coresCfg.AllowAllOrigins = true
		}
		coresCfg.AddAllowHeaders("Authorization")
		g.Use(cors.New(coresCfg))
		streamHandler := web.NewStreamHandler(dbContainer, mediafacade, wp)
		mediaHandler := web.MediaHandler{DBContainer: dbContainer}
		jobReqHandler := web.JobReqHandler{}
		jobResHandler := web.JobResHandler{}
		infoHandler := web.InfoApiHandler{
			MediaFacade: mediafacade,
		}
		loginHandler := web.LoginApiHandler{
			UserName: hCfg.UserName,
			UserPass: hCfg.UserPass,
			Token:    hCfg.ApiToken,
		}
		sessionHandler := web.SessionApiHandler{
			Token: hCfg.ApiToken,
		}
		randomMediaHandler := web.RandomMediaApiHandler{
			MediaFacade: mediafacade,
		}
		hndlrs := web.HandlerContainer{
			MediaHandler:       web.NewCRDApiHandler(&mediaHandler, mediafacade, "media"),
			JobReqHandler:      web.NewCRDApiHandler(&jobReqHandler, jobReqFacade, "jobReq"),
			JobResHandler:      web.NewCRDApiHandler(&jobResHandler, jobResFacade, "jobRes"),
			InfoHandler:        web.NewApiHandler(&infoHandler, "info"),
			LoginHandler:       web.NewApiHandler(&loginHandler, "auth/login"),
			SessionHandler:     web.NewApiHandler(&sessionHandler, "auth/session"),
			RandomMediaHandler: web.NewApiHandler(&randomMediaHandler, "media/random"),
		}
		web.RegisterRoutes(g, streamHandler, hndlrs, hCfg.ApiToken, hCfg.Swagger)
		ll.Warn("starting server")
		if err := g.Run(":8080"); err != nil {
			logrus.WithError(err).Fatal("error running webserver")
		}
	},
}

func init() {
	rootCmd.AddCommand(webCmd)
}
