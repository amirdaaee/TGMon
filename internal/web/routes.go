package web

import (
	"github.com/gin-gonic/gin"
)

func setupRoutes(r *gin.Engine) {
	r.GET("/stream/:mediaID", streamHandler)
	grpApi := r.Group("/api", tokenAuthMiddleware())
	grpApi.GET("/media", listMediaHandler)
	grpApi.DELETE("/media/:mediaID", deleteMediaHandler)
	grpAuth := r.Group("/auth")
	grpAuth.POST("/login", loginHandler)
	grpAuth.GET("/session", sessionHandler, tokenAuthMiddleware())
}
