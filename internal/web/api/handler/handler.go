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
type IPutApiHandler interface {
	Put(g *gin.Context)
	AuthPut() bool
	RelativePathPut() string
}
type IPatchApiHandler interface {
	Patch(g *gin.Context)
	AuthPatch() bool
	RelativePathPatch() string
}
