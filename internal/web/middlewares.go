package web

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Package api provides HTTP API middlewares for authentication, metrics, and error handling.

func apiAuth(c *gin.Context, expected string) bool {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return false
	}
	scheme, value, ok := strings.Cut(authHeader, " ")
	if !ok || !(scheme == "Basic" || scheme == "Bearer") { //nolint:golint,staticcheck
		return false
	}
	return value == expected
}

// apiAuthMiddleware returns a Gin middleware that enforces API authentication using the provided expected value.
func apiAuthMiddleware(expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if expected != "" && !apiAuth(c, expected) {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
		c.Next()
	}
}

// errMiddleware is a Gin middleware that handles errors, logs them, and returns appropriate HTTP responses.
func errMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			switch e := err.Err.(type) {
			case HttpErr:
				c.AbortWithStatusJSON(e.StatusCode, e)
			default:
				c.AbortWithStatusJSON(http.StatusInternalServerError,
					map[string]string{"message": "Service Unavailable"})
			}
		}
	}
}
