package handler

import (
	"math/rand"
	"net/http"

	ftypes "github.com/amirdaaee/TGMon/internal/facade/types"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/types"
	wtypes "github.com/amirdaaee/TGMon/internal/web/types"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type MediaInfoApiHandler struct {
	MediaFacade ftypes.IFacade[types.MediaFileDoc]
}

var _ IGetApiHandler = (*MediaInfoApiHandler)(nil)

// @Summary	Info summary
// @Produce	json
// @Success	200	{object}	InfoGetResType
// @Router		/api/info/ [get]
// @Security	ApiKeyAuth
func (h *MediaInfoApiHandler) Get(g *gin.Context) {
	media, err := h.MediaFacade.GetCollection().Finder().Count(g.Request.Context())
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	g.JSON(http.StatusOK, InfoGetResType{MediaCount: media})
}
func (h *MediaInfoApiHandler) AuthGet() bool {
	return true
}
func (h *MediaInfoApiHandler) RelativePathGet() string {
	return "/"
}

// ===
type RandomMediaApiHandler struct {
	mediaFacade ftypes.IFacade[types.MediaFileDoc]
	ll          *zap.Logger
}

var _ IGetApiHandler = (*RandomMediaApiHandler)(nil)

// @Summary	Get random media
// @Produce	json
// @Success	200	{object}	RandomMediaGetResType
// @Router		/api/media/random/ [get]
// @Security	ApiKeyAuth
func (h *RandomMediaApiHandler) Get(g *gin.Context) {
	ll := h.ll.Named("Get")
	fnd := h.mediaFacade.GetCollection().Finder()
	total, err := fnd.Count(g.Request.Context())
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	n := rand.Int63n(total)
	media, err := fnd.Skip(n).Limit(1).Find(g.Request.Context())
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	ll.Sugar().Debugf("random media: %d. media ID: %s", n, media[0].ID.Hex())
	g.JSON(http.StatusOK, RandomMediaGetResType{MediaID: &media[0].ID})
}
func (h *RandomMediaApiHandler) AuthGet() bool {
	return true
}
func (h *RandomMediaApiHandler) RelativePathGet() string {
	return "/"
}

func NewRandomMediaApiHandler(mediaFacade ftypes.IFacade[types.MediaFileDoc]) *RandomMediaApiHandler {
	return &RandomMediaApiHandler{
		mediaFacade: mediaFacade,
		ll:          log.Named(log.WebModule, "RandomMediaApiHandler"),
	}
}
