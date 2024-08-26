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
	grpApi := r.Group("/api", tokenAuthMiddleware())
	grpApi.GET("/media", listMediaHandlerFactory(mongo))
	grpApi.DELETE("/media/:mediaID", deleteMediaHandlerFactory(wp, mongo, minio))
	// ...
	cfg := config.Config()
	grpAuth := r.Group("/auth")
	grpAuth.POST("/login", loginHandlerFactory(cfg.UserName, cfg.UserPass, cfg.UserToken))
	grpAuth.GET("/session", sessionHandlerFactory(cfg.UserToken), tokenAuthMiddleware())
}
