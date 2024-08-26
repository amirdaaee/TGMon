package main

import (
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
	r := gin.Default()
	web.SetupRoutes(r, wp, mongo, minio, config.Config().StreamChunkSize)
	if err := r.Run(); err != nil {
		logrus.WithError(err).Error("server terminated")
	}
}
