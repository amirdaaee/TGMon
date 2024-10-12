package web

import (
	"net/http"

	"github.com/amirdaaee/TGMon/config"
	"github.com/gin-gonic/gin"
)

func tokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := c.Request.Header["Authorization"]
		if !ok || token[0] != "Bearer "+config.Config().UserToken {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
		c.Next()
	}
}
func apiAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := c.Request.Header["Authorization"]

		if !ok || token[0] != "Basic "+config.Config().ApiToken {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
		c.Next()
	}
}
