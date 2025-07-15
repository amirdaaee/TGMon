package web

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type ApiHandler struct {
	hndler IApiHandler
	name   string
}

func (a *ApiHandler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	apiG := r.Group(fmt.Sprintf("/%s", a.name))

	if a.hndler.HasPost() != No {
		mid := []gin.HandlerFunc{}
		if a.hndler.HasPost() == Auth {
			mid = append(mid, authMiddleware)
		}
		mid = append(mid, a.hndler.Post)
		apiG.POST("/", mid...)
	}
	if a.hndler.HasGet() != No {
		mid := []gin.HandlerFunc{}
		if a.hndler.HasGet() == Auth {
			mid = append(mid, authMiddleware)
		}
		mid = append(mid, a.hndler.Get)
		apiG.GET("/", mid...)
	}
}

func NewApiHandler(hndler IApiHandler, name string) *ApiHandler {
	return &ApiHandler{hndler: hndler, name: name}
}
