/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/amirdaaee/TGMon/internal/app"
	"github.com/amirdaaee/TGMon/internal/config"
	"github.com/amirdaaee/TGMon/internal/filesystem"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start TGmon web and fuse server",
	Run: func(cmd *cobra.Command, args []string) {
		ll := logrus.WithField("at", "web")
		ll.Info("starting web server")
		cfg := config.Config()
		//...
		ctx, cancl := context.WithCancel(context.TODO())
		defer cancl()
		eg, ctx := errgroup.WithContext(ctx)
		//...
		servers, err := app.InitializeServer(cfg)
		if err != nil {
			logrus.WithError(err).Fatal("can not initialize servers")
		}
		webServer := servers.WebServer
		fuseServer := servers.FuzeServer
		eg.Go(func() error {
			if err := webServer.ListenAndServe(); err != nil {
				return fmt.Errorf("error running webserver: %w", err)
			}
			ll.Info("server stopped")
			return nil
		})
		eg.Go(func() error {
			<-ctx.Done()
			ll.Info("shutting down web server")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := webServer.Shutdown(ctx); err != nil {
				ll.WithError(err).Error("error shutting down web server")
			}
			ll.Info("web server shutdown")
			return nil
		})
		// ...
		if cfg.FuseConfig.Enabled {
			ll.Info("fuse server started")
			eg.Go(func() error {
				<-ctx.Done()
				ll.Info("shutting down fuse server")
				if err := fuseServer.Unmount(); err != nil {
					ll.WithError(err).Error("error unmounting fuse server (method 1)")
					if err := filesystem.Unmount(cfg.FuseConfig.MediaDir); err != nil {
						ll.WithError(err).Error("error unmounting fuse server (method 2)")
					} else {
						ll.Info("fuse server unmounted")
					}
				} else {
					ll.Info("fuse server unmounted")
				}
				return nil
			})
		}
		// ...
		eg.Go(func() error {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			select {
			case sig := <-sigChan:
				return fmt.Errorf("received signal to stop server: %s", sig)
			case <-ctx.Done():
				return nil
			}
		})
		// ...
		if err := eg.Wait(); err != nil && !strings.HasPrefix(err.Error(), "received signal to stop server") {
			logrus.WithError(err).Error("error in server")
		} else {
			ll.Info("web server stopped")
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
