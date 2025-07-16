package web

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type ApiHandler struct {
	hndler any
	name   string
}

func (a *ApiHandler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	apiG := r.Group(fmt.Sprintf("/%s", a.name))
	if v, ok := a.hndler.(IPostApiHandler); ok {
		mid := []gin.HandlerFunc{}
		if v.AuthPost() {
			mid = append(mid, authMiddleware)
		}
		mid = append(mid, v.Post)
		apiG.POST("/", mid...)
	}
	if v, ok := a.hndler.(IGetApiHandler); ok {
		mid := []gin.HandlerFunc{}
		if v.AuthGet() {
			mid = append(mid, authMiddleware)
		}
		mid = append(mid, v.Get)
		apiG.GET("/", mid...)
	}
}

func NewApiHandler(hndler any, name string) *ApiHandler {
	return &ApiHandler{hndler: hndler, name: name}
}
