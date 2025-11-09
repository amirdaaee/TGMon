package web

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"github.com/amirdaaee/TGMon/internal/facade"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/stash"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2/bsonx"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
)

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
type StashVTTRedirectorApiHandler struct {
	MinioUrl    string
	StashCl     *stash.StashQlClient
	MediaFacade facade.IFacade[types.MediaFileDoc]
}
type StashCoverRedirectorApiHandler struct {
	StashVTTRedirectorApiHandler
}

var _ IGetApiHandler = (*InfoApiHandler)(nil)
var _ IGetApiHandler = (*SessionApiHandler)(nil)
var _ IGetApiHandler = (*RandomMediaApiHandler)(nil)
var _ IPostApiHandler = (*LoginApiHandler)(nil)
var _ IGetApiHandler = (*StashVTTRedirectorApiHandler)(nil)
var _ IGetApiHandler = (*StashCoverRedirectorApiHandler)(nil)

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
func (h *InfoApiHandler) RelativePathGet() string {
	return "/"
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
func (h *LoginApiHandler) RelativePathPost() string {
	return "/"
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
func (h *SessionApiHandler) RelativePathGet() string {
	return "/"
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
func (h *RandomMediaApiHandler) RelativePathGet() string {
	return "/"
}

// getLogger returns a logger entry with function context for the Bot.
func (h *RandomMediaApiHandler) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.WebModule).WithField("func", fmt.Sprintf("%T.%s", h, fn))
}

// ===
type idURIType struct {
	ID string `uri:"id" binding:"required"`
}

func (h *StashVTTRedirectorApiHandler) Get(g *gin.Context) {
	var id idURIType
	if err := g.ShouldBindUri(&id); err != nil {
		g.Error(NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	osHashSplt := strings.Split(id.ID, "_")
	scene, err := h.StashCl.FindSceneByHash(g.Request.Context(), osHashSplt[0])
	if err != nil {
		g.Error(NewHttpError(err, http.StatusNotFound)) //nolint:golint,errcheck
		return
	}
	media, err := h.getMediaByScene(g.Request.Context(), scene)
	if err != nil {
		g.Error(NewHttpError(err, http.StatusNotFound)) //nolint:golint,errcheck
		return
	}
	if media != nil && media.Vtt != "" {
		g.Redirect(http.StatusPermanentRedirect, fmt.Sprintf("%s/%s", h.MinioUrl, media.Vtt))

	} else {
		g.Error(NewHttpError(errors.New("no vtt file found"), http.StatusNotFound)) //nolint:golint,errcheck
	}
}
func (h *StashVTTRedirectorApiHandler) AuthGet() bool {
	return false
}
func (h *StashVTTRedirectorApiHandler) RelativePathGet() string {
	return "/scene/:id"
}

// func (h *StashVTTRedirectorApiHandler) getLogger(fn string) *logrus.Entry {
// 	return log.GetLogger(log.WebModule).WithField("func", fmt.Sprintf("%T.%s", h, fn))
// }

func (h *StashVTTRedirectorApiHandler) getMediaByScene(ctx context.Context, scene *stash.Scene) (*types.MediaFileDoc, error) {
	fname := scene.Files[0].Basename
	fIDSplit := strings.Split(fname, "-")
	fID := strings.Split(fIDSplit[len(fIDSplit)-1], ".")[0]
	mongoID, err := bson.ObjectIDFromHex(fID)
	if err != nil {
		return nil, fmt.Errorf("can not parse mongo id: %w", err)
	}
	media, err := h.MediaFacade.GetCollection().Finder().Filter(bsonx.Id(mongoID)).FindOne(ctx)
	if err != nil {
		return nil, fmt.Errorf("can not query media by mongo id (%s): %w", mongoID, err)
	}
	return media, nil
}

// ===
func (h *StashCoverRedirectorApiHandler) Get(g *gin.Context) {
	var id idURIType
	if err := g.ShouldBindUri(&id); err != nil {
		g.Error(NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	scene, err := h.StashCl.FindSceneById(g.Request.Context(), id.ID)
	if err != nil {
		g.Error(NewHttpError(err, http.StatusNotFound)) //nolint:golint,errcheck
		return
	}
	media, err := h.getMediaByScene(g.Request.Context(), scene)
	if err != nil {
		g.Error(NewHttpError(err, http.StatusNotFound)) //nolint:golint,errcheck
		return
	}
	if media != nil {
		g.Redirect(http.StatusPermanentRedirect, fmt.Sprintf("%s/%s", h.MinioUrl, media.Thumbnail))

	} else {
		g.Error(NewHttpError(err, http.StatusNotFound)) //nolint:golint,errcheck
	}
}
func (h *StashCoverRedirectorApiHandler) RelativePathGet() string {
	return "/scene/:id/screenshot"
}
