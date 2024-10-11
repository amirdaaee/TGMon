package web

import (
	"github.com/amirdaaee/TGMon/config"
	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, wp *bot.WorkerPool, mongo *db.Mongo, minio *db.MinioClient, cfg *config.ConfigType) {
	r.Match([]string{"HEAD", "GET"}, "/stream/:mediaID", streamHandlerFactory(wp, mongo, cfg.StreamChunkSize, cfg.WorkerProfileFile))
	// ...
	grpApi := r.Group("/api/media")
	grpApi.GET("/", tokenAuthMiddleware(), listMediaHandlerFactory(mongo))
	grpApi.GET("/:mediaID", infoMediaHandlerFactory(mongo))
	grpApi.DELETE("/:mediaID", deleteMediaHandlerFactory(wp, mongo, minio))
	grpApi.POST("/thumbgen", tokenAuthMiddleware(), createThumbnailHandlerFactory(mongo, minio, cfg.FFmpegImage, cfg.ServerURL))
	// ...
	grpAuth := r.Group("/api/auth")
	grpAuth.POST("/login", loginHandlerFactory(cfg.UserName, cfg.UserPass, cfg.UserToken))
	grpAuth.GET("/session", tokenAuthMiddleware(), sessionHandlerFactory(cfg.UserToken))
}
