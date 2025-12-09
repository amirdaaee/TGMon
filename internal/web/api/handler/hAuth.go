package handler

import (
	"errors"
	"net/http"

	wtypes "github.com/amirdaaee/TGMon/internal/web/types"
	"github.com/gin-gonic/gin"
)

type LoginApiHandler struct {
	UserName string
	UserPass string
	Token    string
}

var _ IPostApiHandler = (*LoginApiHandler)(nil)

// @Summary      Login
// @Description  Authenticate user and return a token
// @Accept       json
// @Produce      json
// @Param        data  body      LoginPostReqType  true  "Login Data"
// @Router       /api/auth/login/ [post]
// @Success	200	{object}	LoginPostResType
func (h *LoginApiHandler) Post(g *gin.Context) {
	var req LoginPostReqType
	if err := g.ShouldBindJSON(&req); err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	if req.Username != h.UserName || req.Password != h.UserPass {
		g.Error(wtypes.NewHttpError(errors.New("invalid username or password"), http.StatusUnauthorized)) //nolint:golint,errcheck
		return
	}
	g.JSON(http.StatusOK, LoginPostResType{Token: h.Token})
}
func (h *LoginApiHandler) AuthPost() bool {
	return false
}
func (h *LoginApiHandler) RelativePathPost() string {
	return "/"
}

// ===
type SessionApiHandler struct {
	Token string
}

var _ IGetApiHandler = (*SessionApiHandler)(nil)

// @Summary	Session data
// @Produce	json
// @Router		/api/auth/session/ [get]
// @Security	ApiKeyAuth
// @Success	200	{object}	LoginPostResType
func (h *SessionApiHandler) Get(g *gin.Context) {
	g.JSON(http.StatusOK, LoginPostResType{Token: h.Token})
}
func (h *SessionApiHandler) AuthGet() bool {
	return true
}
func (h *SessionApiHandler) RelativePathGet() string {
	return "/"
}
