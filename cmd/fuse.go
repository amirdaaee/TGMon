/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/amirdaaee/TGMon/internal/filesystem"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// fuseCmd represents the fuse command
var fuseCmd = &cobra.Command{
	Use:   "fuse",
	Short: "Mount media as file system",
	Run: func(cmd *cobra.Command, args []string) {
		setupLogger()
		ll := logrus.WithField("at", "fuse")
		ll.Info("starting fuse server")
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
		mountDir := cmd.Flag("dir").Value.String()
		allowOther, _ := cmd.Flags().GetBool("allow-other")
		debug, _ := cmd.Flags().GetBool("debug")

		opts := &filesystem.MountOptions{
			AllowOther: allowOther,
			Debug:      debug,
		}

		server, err := filesystem.MountWithOptions(mountDir, dbContainer, wp, opts)
		if err != nil {
			logrus.WithError(err).Fatal("can not mount filesystem")
		}
		ll.Info("fuse server started")
		func() {
			defer func() {
				if err := server.Unmount(); err != nil {
					logrus.WithError(err).Error("can not unmount filesystem")
				}
				ll.Info("fuse server stopped")
			}()
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
			<-sig
		}()
	},
}

func init() {
	rootCmd.AddCommand(fuseCmd)
	fuseCmd.Flags().StringP("dir", "d", "./storage/media", "Directory to mount")
	fuseCmd.Flags().BoolP("allow-other", "o", false, "Allow other users/containers to access the filesystem (requires user_allow_other in /etc/fuse.conf)")
	fuseCmd.Flags().BoolP("debug", "v", false, "Enable FUSE debug logging")
}
