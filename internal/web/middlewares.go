package web

import (
	"net/http"

	"github.com/amirdaaee/TGMon/config"
	"github.com/gin-gonic/gin"
)

func tokenAuth(c *gin.Context) bool {
	token, ok := c.Request.Header["Authorization"]
	if !ok || token[0] != "Bearer "+config.Config().UserToken {
		return false
	}
	return true
}
func apiAuth(c *gin.Context) bool {
	token, ok := c.Request.Header["Authorization"]
	if !ok || token[0] != "Basic "+config.Config().ApiToken {
		return false
	}
	return true
}
func tokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !tokenAuth(c) {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
		c.Next()
	}
}
func apiAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !apiAuth(c) {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
		c.Next()
	}
}

func anyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !(apiAuth(c) || tokenAuth(c)) {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
		c.Next()
	}
}
