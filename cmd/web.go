/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amirdaaee/TGMon/internal/config"
	"github.com/amirdaaee/TGMon/internal/filesystem"
	"github.com/amirdaaee/TGMon/internal/web"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
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
		wp, err := buildWorkerPool()
		if err != nil {
			logrus.WithError(err).Fatal("can not build worker pool")
		}
		// ...
		mediafacade := buildMediaFacade(dbContainer, wp)
		jobReqFacade := buildJobReqFacade(dbContainer)
		jobResFacade := buildJobResFacade(dbContainer)
		ll.Info("media facade built")
		// ...
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		errG, ctx := errgroup.WithContext(ctx)
		// ...
		fCfg := config.Config().FuseConfig
		var fuseServer *fuse.Server
		if fCfg.Enabled {
			errG.Go(func() error {
				ll.Info("fuse config enabled")
				mountDir := fCfg.MediaDir
				opts := &filesystem.MountOptions{
					AllowOther: fCfg.AllowOther,
					Debug:      fCfg.Debug,
				}
				ll.Info("starting fuse server")
				server, err := filesystem.MountWithOptions(mountDir, dbContainer, wp, opts)
				if err != nil {
					return fmt.Errorf("can not mount filesystem: %w", err)
				}
				fuseServer = server
				ll.Info("fuse server started")
				return nil
			})
		}
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
		srv := &http.Server{
			Addr:    hCfg.ListenAddr,
			Handler: g,
		}
		errG.Go(func() error {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				return fmt.Errorf("error running webserver: %w", err)
			}
			ll.Info("server stopped")
			return nil
		})
		errG.Go(func() error {
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
			select {
			case <-quit:
				return fmt.Errorf("received shutdown signal: %d", syscall.SIGINT)
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		errG.Go(func() error {
			<-ctx.Done()
			if fuseServer == nil {
				ll.Info("fuse server not mounted")
			} else {
				if err := fuseServer.Unmount(); err != nil {
					logrus.WithError(err).Error("can not unmount filesystem")
				} else {
					ll.Info("fuse server stopped")
				}
			}
			// ...
			shutCtx, shutCtxFn := context.WithTimeout(context.TODO(), 10*time.Second)
			defer shutCtxFn()
			if err := srv.Shutdown(shutCtx); err != nil {
				logrus.WithError(err).Error("can not shutdown server")
			} else {
				ll.Info("server stopped")
			}
			return nil
		})
		// ...
		if err := errG.Wait(); err != nil {
			logrus.WithError(err).Error("error in server")
		} else {
			ll.Info("web server stopped")
		}
	},
}

func init() {
	rootCmd.AddCommand(webCmd)
}
