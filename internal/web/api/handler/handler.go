package handler

import "github.com/gin-gonic/gin"

type IPostApiHandler interface {
	Post(g *gin.Context)
	AuthPost() bool
	RelativePathPost() string
}
type IGetApiHandler interface {
	Get(g *gin.Context)
	AuthGet() bool
	RelativePathGet() string
}
