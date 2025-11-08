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

	"github.com/amirdaaee/TGMon/internal/config"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/facade"
	"github.com/amirdaaee/TGMon/internal/filesystem"
	"github.com/amirdaaee/TGMon/internal/stream"
	"github.com/amirdaaee/TGMon/internal/types"
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

		// ...
		webStopper, err := webServerHandler(dbContainer, mediafacade, wp, jobReqFacade, jobResFacade, errG)
		if err != nil {
			logrus.WithError(err).Fatal("can not start web server")
		}
		defer func() {
			if webStopper != nil {
				if err := webStopper(); err != nil {
					logrus.Error(err)
				}
			}
		}()
		// ...
		fuseStopper, err := fuseServerHandler(dbContainer, wp, errG)
		if err != nil {
			logrus.WithError(err).Fatal("can not start fuse server")
		}
		defer func() {
			if fuseStopper != nil {
				if err := fuseStopper(); err != nil {
					logrus.Error(err)
				}
			}
		}()
		// ...
		errG.Go(func() error {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			sig := <-sigChan
			return fmt.Errorf("received signal to stop server: %s", sig) // to stop error group
		})
		errG.Go(func() error {
			<-ctx.Done()
			if webStopper != nil {
				if err := webStopper(); err != nil {
					logrus.Error(err)
				}
			}
			if fuseStopper != nil {
				if err := fuseStopper(); err != nil {
					logrus.Error(err)
				}
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

type Stopper func() error

func webServerHandler(dbContainer db.IDbContainer, mediafacade facade.IFacade[types.MediaFileDoc], wp stream.IWorkerPool, jobReqFacade facade.IFacade[types.JobReqDoc], jobResFacade facade.IFacade[types.JobResDoc], errG *errgroup.Group) (Stopper, error) {
	ll := logrus.WithField("at", "webServerHandler")
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
	return func() error {
		if err := srv.Shutdown(context.TODO()); err != nil {
			return fmt.Errorf("can not shutdown server: %w", err)
		}
		ll.Info("web server stopped")
		return nil
	}, nil
}

func fuseServerHandler(dbContainer db.IDbContainer, wp stream.IWorkerPool, errG *errgroup.Group) (Stopper, error) {
	ll := logrus.WithField("at", "fuseServerHandler")
	fCfg := config.Config().FuseConfig
	if fCfg.Enabled {
		var fuseSrv *fuse.Server
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
			fuseSrv = server
			ll.Info("fuse server started")
			return nil
		})
		return func() error {
			if err := fuseSrv.Unmount(); err != nil {
				return fmt.Errorf("can not unmount filesystem: %w", err)
			}
			ll.Info("fuse server stopped")
			return nil
		}, nil
	}
	return nil, nil
}
