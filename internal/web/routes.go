package web

import (
	"github.com/amirdaaee/TGMon/config"
	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, wp *bot.WorkerPool, mongo *db.Mongo, minio db.IMinioClient, cfg *config.ConfigType) {
	r.Match([]string{"HEAD", "GET"}, "/stream/:mediaID", streamHandlerFactory(wp, mongo, cfg.StreamChunkSize, cfg.WorkerProfileFile))
	// ...
	mediaApi := r.Group("/api/media")
	mediaApi.GET("/", tokenAuthMiddleware(), listMediaHandlerFactory(mongo))
	mediaApi.GET("/rand", tokenAuthMiddleware(), getRandomMedia(mongo))
	mediaApi.GET("/:mediaID", infoMediaHandlerFactory(mongo))
	mediaApi.DELETE("/:mediaID", deleteMediaHandlerFactory(wp, mongo, minio))
	// ...
	jobApi := r.Group("/api/job")
	jobApi.GET("/", anyAuthMiddleware(), listJobsHandlerFactory(mongo))
	jobApi.POST("/", tokenAuthMiddleware(), createJobsHandlerFactory(mongo))
	jobApi.PUT("/result/:jobID/:status", apiAuthMiddleware(), putJobResultHandlerFactory(mongo, minio))

	// ...
	grpAuth := r.Group("/api/auth")
	grpAuth.POST("/login", loginHandlerFactory(cfg.UserName, cfg.UserPass, cfg.UserToken))
	grpAuth.GET("/session", tokenAuthMiddleware(), sessionHandlerFactory(cfg.UserToken))
}
