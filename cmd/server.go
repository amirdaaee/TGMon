/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/amirdaaee/TGMon/internal/db"
	fsSrc "github.com/amirdaaee/TGMon/internal/filesystem/src"

	"github.com/amirdaaee/TGMon/cmd/wire"
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
		cntr := wire.GetProvider()
		//...
		ctx, cancl := context.WithCancel(cmd.Context())
		defer cancl()
		eg, ctx := errgroup.WithContext(ctx)
		//...
		if err := cntr.Invoke(func(webServer *http.Server, cfg *config.ConfigType, dbC db.IDbContainer, srcs []fsSrc.ISrc) error {
			eg.Go(func() error {
				if err := webServer.ListenAndServe(); err != nil {
					if err == http.ErrServerClosed {
						return nil
					}
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
				eg.Go(func() error {
					fCfg := cfg.FuseConfig
					mountDir := fCfg.MediaDir
					opts := &filesystem.MountOptions{
						AllowOther: fCfg.AllowOther,
						Debug:      fCfg.Debug,
					}
					fuseServer, err := filesystem.MountWithOptions(mountDir, srcs, dbC, opts)
					if err != nil {
						return fmt.Errorf("can not mount filesystem: %w", err)
					}
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
			if err := eg.Wait(); err != nil && !strings.HasPrefix(err.Error(), "received signal to stop server") {
				return fmt.Errorf("error in server: %w", err)
			}
			ll.Info("web server stopped")
			return nil
		}); err != nil {
			ll.WithError(err).Error("error in server")
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
