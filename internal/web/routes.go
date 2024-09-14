package web

import (
	"github.com/amirdaaee/TGMon/config"
	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, wp *bot.WorkerPool, mongo *db.Mongo, minio *db.MinioClient, streamChunckSize int64) {
	r.GET("/stream/:mediaID", streamHandlerFactory(wp, mongo, streamChunckSize))
	// ...
	grpApi := r.Group("/api/media", tokenAuthMiddleware())
	grpApi.GET("/", listMediaHandlerFactory(mongo))
	grpApi.DELETE("/:mediaID", deleteMediaHandlerFactory(wp, mongo, minio))
	// ...
	cfg := config.Config()
	grpAuth := r.Group("/api/auth")
	grpAuth.POST("/login", loginHandlerFactory(cfg.UserName, cfg.UserPass, cfg.UserToken))
	grpAuth.GET("/session", tokenAuthMiddleware(), sessionHandlerFactory(cfg.UserToken))
}
