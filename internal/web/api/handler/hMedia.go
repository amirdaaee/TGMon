package handler

import (
	"errors"
	"net/http"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/repository"
	wtypes "github.com/amirdaaee/TGMon/internal/web/types"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type MediaInfoApiHandler struct {
	Media repository.MediaFileRepository
}

var _ IGetApiHandler = (*MediaInfoApiHandler)(nil)

// @Summary	Info summary
// @Produce	json
// @Success	200	{object}	InfoGetResType
// @Router		/api/info/ [get]
// @Security	ApiKeyAuth
func (h *MediaInfoApiHandler) Get(g *gin.Context) {
	media, err := h.Media.Count(g.Request.Context())
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
	media repository.MediaFileRepository
	ll    *zap.Logger
}

var _ IGetApiHandler = (*RandomMediaApiHandler)(nil)

// @Summary	Get random media
// @Produce	json
// @Success	200	{object}	RandomMediaGetResType
// @Router		/api/media/random/ [get]
// @Security	ApiKeyAuth
func (h *RandomMediaApiHandler) Get(g *gin.Context) {
	ll := h.ll.Named("Get")
	media, err := h.media.FindRandom(g.Request.Context())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			g.Error(wtypes.NewHttpError(err, http.StatusNotFound)) //nolint:golint,errcheck
			return
		}
		g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	ll.Sugar().Debugf("random media ID: %s", media.ID.Hex())
	g.JSON(http.StatusOK, RandomMediaGetResType{MediaID: &media.ID})
}
func (h *RandomMediaApiHandler) AuthGet() bool {
	return true
}
func (h *RandomMediaApiHandler) RelativePathGet() string {
	return "/"
}

func NewRandomMediaApiHandler(media repository.MediaFileRepository) *RandomMediaApiHandler {
	return &RandomMediaApiHandler{
		media: media,
		ll:    log.Named(log.WebModule, "RandomMediaApiHandler"),
	}
}
