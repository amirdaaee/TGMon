package api

import (
	"fmt"

	"github.com/amirdaaee/TGMon/internal/web/api/handler"
	wtypes "github.com/amirdaaee/TGMon/internal/web/types"
	"github.com/gin-gonic/gin"
)

type ApiHandler struct {
	hndler any
	name   string
}

var _ wtypes.Registereable = (*ApiHandler)(nil)

func (a *ApiHandler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) error {
	apiG := r.Group(fmt.Sprintf("/%s", a.name))
	registered := false
	if v, ok := a.hndler.(handler.IPostApiHandler); ok {
		mid := []gin.HandlerFunc{}
		if v.AuthPost() {
			mid = append(mid, authMiddleware)
		}
		mid = append(mid, v.Post)
		apiG.POST(v.RelativePathPost(), mid...)
		registered = true
	}
	if v, ok := a.hndler.(handler.IGetApiHandler); ok {
		mid := []gin.HandlerFunc{}
		if v.AuthGet() {
			mid = append(mid, authMiddleware)
		}
		mid = append(mid, v.Get)
		apiG.GET(v.RelativePathGet(), mid...)
		registered = true
	}
	if !registered {
		return fmt.Errorf("no routes registered for %s", a.name)
	}
	return nil
}

func (a *ApiHandler) RegisterToRoot() bool {
	if v, ok := a.hndler.(wtypes.RootRegisterer); !ok {
		return false
	} else {
		return v.RegisterToRoot()
	}
}

func NewApiHandler(hndler any, name string) *ApiHandler {
	return &ApiHandler{hndler: hndler, name: name}
}
