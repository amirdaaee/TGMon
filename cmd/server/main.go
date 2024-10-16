package main

import (
	"io"
	"os"

	"github.com/amirdaaee/TGMon/cmd"
	"github.com/amirdaaee/TGMon/config"
	"github.com/amirdaaee/TGMon/internal/web"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	wp, err := cmd.GetWorkerPool()
	if err != nil {
		logrus.WithError(err).Fatal("can not start workers")
	}
	mongo := cmd.GetMongoDB()
	minio, err := cmd.GetMinioDB()
	if err != nil {
		logrus.WithError(err).Fatal("can not create minio client")
	}
	// ...
	accLogFile := config.Config().AccessLogFile
	if accLogFile != "" {
		gin.DisableConsoleColor()
		if f, err := os.Create(accLogFile); err != nil {
			logrus.WithError(err).Fatalf("can not create access log file at %s", accLogFile)
		} else {
			gin.DefaultWriter = io.MultiWriter(f)
			logrus.Infof("access log file at %s", accLogFile)
		}
	}
	// ...
	r := gin.Default()
	web.SetupRoutes(r, wp, mongo, minio, config.Config())
	// ...
	if err := r.Run(); err != nil {
		logrus.WithError(err).Error("server terminated")
	}
}
