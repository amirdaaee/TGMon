package web

import (
	"errors"
	"net/http"

	"github.com/amirdaaee/TGMon/internal/facade"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/gin-gonic/gin"
)

type IApiHandler interface {
	Post(g *gin.Context)
	Get(g *gin.Context)
	HasPost() ApiType
	HasGet() ApiType
}

type InfoApiHandler struct {
	MediaFacade facade.IFacade[types.MediaFileDoc]
}
type LoginApiHandler struct {
	UserName string
	UserPass string
	Token    string
}

var _ IApiHandler = (*InfoApiHandler)(nil)
var _ IApiHandler = (*LoginApiHandler)(nil)

func (h *InfoApiHandler) Post(g *gin.Context) {}
func (h *InfoApiHandler) Get(g *gin.Context) {
	media, err := h.MediaFacade.GetCollection().Finder().Count(g.Request.Context())
	if err != nil {
		g.Error(NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	g.JSON(http.StatusOK, InfoGetResType{MediaCount: media})
}
func (h *InfoApiHandler) HasPost() ApiType {
	return No
}
func (h *InfoApiHandler) HasGet() ApiType {
	return Auth
}

// ===
func (h *LoginApiHandler) Post(g *gin.Context) {
	var req LoginPostReqType
	if err := g.ShouldBindJSON(&req); err != nil {
		g.Error(NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	if req.Username != h.UserName || req.Password != h.UserPass {
		g.Error(NewHttpError(errors.New("invalid username or password"), http.StatusUnauthorized)) //nolint:golint,errcheck
		return
	}
	g.JSON(http.StatusOK, LoginPostResType{Token: h.Token})
}
func (h *LoginApiHandler) Get(g *gin.Context) {}
func (h *LoginApiHandler) HasPost() ApiType {
	return NoAuth
}
func (h *LoginApiHandler) HasGet() ApiType {
	return No
}
