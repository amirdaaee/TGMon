package web

import (
	"errors"
	"net/http"

	"github.com/amirdaaee/TGMon/internal/facade"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/gin-gonic/gin"
)

type IPostApiHandler interface {
	Post(g *gin.Context)
	AuthPost() bool
}
type IGetApiHandler interface {
	Get(g *gin.Context)
	AuthGet() bool
}

type InfoApiHandler struct {
	MediaFacade facade.IFacade[types.MediaFileDoc]
}
type LoginApiHandler struct {
	UserName string
	UserPass string
	Token    string
}

var _ IGetApiHandler = (*InfoApiHandler)(nil)
var _ IPostApiHandler = (*LoginApiHandler)(nil)

// @Summary	Info summary
// @Produce	json
// @Success	200	{object}	InfoGetResType
// @Router		/api/info [get]
// @Security	ApiKeyAuth
func (h *InfoApiHandler) Get(g *gin.Context) {
	media, err := h.MediaFacade.GetCollection().Finder().Count(g.Request.Context())
	if err != nil {
		g.Error(NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	g.JSON(http.StatusOK, InfoGetResType{MediaCount: media})
}
func (h *InfoApiHandler) AuthGet() bool {
	return true
}

// ===
// @Summary      Login
// @Description  Authenticate user and return a token
// @Accept       json
// @Produce      json
// @Param        data  body      LoginPostReqType  true  "Login Data"
// @Router       /api/login [post]
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
func (h *LoginApiHandler) AuthPost() bool {
	return false
}
