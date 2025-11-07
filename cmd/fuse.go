/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
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
		wp, err := buildWorkerContainer()
		if err != nil {
			logrus.WithError(err).Fatal("can not build worker pool")
		}
		// ...
		mountDir := cmd.Flag("dir").Value.String()
		server, err := filesystem.Mount(mountDir, dbContainer, wp)
		if err != nil {
			logrus.WithError(err).Fatal("can not mount filesystem")
		}
		ll.Info("fuse server started")
		defer func() {
			if err := server.Unmount(); err != nil {
				logrus.WithError(err).Error("can not unmount filesystem")
			}
		}()
		ll.Info("fuse server stopped")
		select {}
	},
}

func init() {
	rootCmd.AddCommand(fuseCmd)
	fuseCmd.Flags().StringP("dir", "d", "./storage/media", "Directory to mount")
}
