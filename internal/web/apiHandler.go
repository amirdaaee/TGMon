package web

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/amirdaaee/TGMon/internal/facade"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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
type SessionApiHandler struct {
	Token string
}
type RandomMediaApiHandler struct {
	MediaFacade facade.IFacade[types.MediaFileDoc]
}

var _ IGetApiHandler = (*InfoApiHandler)(nil)
var _ IGetApiHandler = (*SessionApiHandler)(nil)
var _ IGetApiHandler = (*RandomMediaApiHandler)(nil)
var _ IPostApiHandler = (*LoginApiHandler)(nil)

// @Summary	Info summary
// @Produce	json
// @Success	200	{object}	InfoGetResType
// @Router		/api/info/ [get]
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
// @Router       /api/auth/login/ [post]
// @Success	200	{object}	LoginPostResType
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

// ===
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

// ===
// @Summary	Get random media
// @Produce	json
// @Success	200	{object}	RandomMediaGetResType
// @Router		/api/media/random/ [get]
// @Security	ApiKeyAuth
func (h *RandomMediaApiHandler) Get(g *gin.Context) {
	ll := h.getLogger("Get")
	fnd := h.MediaFacade.GetCollection().Finder()
	total, err := fnd.Count(g.Request.Context())
	if err != nil {
		g.Error(NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	n := rand.Int63n(total)
	media, err := fnd.Skip(n).Limit(1).Find(g.Request.Context())
	if err != nil {
		g.Error(NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	ll.Infof("random media: %d. media ID: %s", n, media[0].ID.Hex())
	g.JSON(http.StatusOK, RandomMediaGetResType{MediaID: &media[0].ID})
}
func (h *RandomMediaApiHandler) AuthGet() bool {
	return true
}

// getLogger returns a logger entry with function context for the Bot.
func (h *RandomMediaApiHandler) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.WebModule).WithField("func", fmt.Sprintf("%T.%s", h, fn))
}
