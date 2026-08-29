package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/amirdaaee/TGMon/internal/repository"
	"github.com/amirdaaee/TGMon/internal/types"
	wtypes "github.com/amirdaaee/TGMon/internal/web/types"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type MediaMetaApiHandler struct {
	media repository.MediaFileRepository
	meta  repository.MediaExtendedMetaRepository
}

var _ IGetApiHandler = (*MediaMetaApiHandler)(nil)
var _ IPutApiHandler = (*MediaMetaApiHandler)(nil)
var _ IPatchApiHandler = (*MediaMetaApiHandler)(nil)

func NewMediaMetaApiHandler(media repository.MediaFileRepository, meta repository.MediaExtendedMetaRepository) *MediaMetaApiHandler {
	return &MediaMetaApiHandler{
		media: media,
		meta:  meta,
	}
}

// @Summary	Get media extended meta
// @Tags		mediaMeta
// @Produce	json
// @Param		id	path	string	true	"Media ID"
// @Success	200	{object}	types.MediaExtendedMeta
// @Router		/api/media/{id}/meta/ [get]
// @Security	ApiKeyAuth
func (h *MediaMetaApiHandler) Get(g *gin.Context) {
	id, ok := h.bindMediaID(g)
	if !ok {
		return
	}
	if !h.requireMedia(g, id) {
		return
	}
	doc, err := h.meta.GetOrCreateByMediaFileID(g.Request.Context(), id)
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	g.JSON(http.StatusOK, doc)
}
func (h *MediaMetaApiHandler) AuthGet() bool {
	return true
}
func (h *MediaMetaApiHandler) RelativePathGet() string {
	return "/"
}

// @Summary	Replace media extended meta
// @Tags		mediaMeta
// @Accept		json
// @Produce	json
// @Param		id		path	string				true	"Media ID"
// @Param		data	body	MediaMetaPutReqType	true	"Media meta"
// @Success	200		{object}	types.MediaExtendedMeta
// @Router		/api/media/{id}/meta/ [put]
// @Security	ApiKeyAuth
func (h *MediaMetaApiHandler) Put(g *gin.Context) {
	id, ok := h.bindMediaID(g)
	if !ok {
		return
	}
	if !h.requireMedia(g, id) {
		return
	}
	var req MediaMetaPutReqType
	if err := g.ShouldBindJSON(&req); err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	doc, err := h.meta.ReplaceByMediaFileID(g.Request.Context(), id, &types.MediaExtendedMeta{
		LastPlayedAt: req.LastPlayedAt,
		Checkpoint:   req.Checkpoint,
		Score:        req.Score,
		Likes:        req.Likes,
		IsFavorite:   req.IsFavorite,
	})
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	g.JSON(http.StatusOK, doc)
}
func (h *MediaMetaApiHandler) AuthPut() bool {
	return true
}
func (h *MediaMetaApiHandler) RelativePathPut() string {
	return "/"
}

// @Summary	Patch media extended meta
// @Tags		mediaMeta
// @Accept		json
// @Produce	json
// @Param		id		path	string					true	"Media ID"
// @Param		data	body	MediaMetaPatchReqType	true	"Media meta patch"
// @Success	200		{object}	types.MediaExtendedMeta
// @Router		/api/media/{id}/meta/ [patch]
// @Security	ApiKeyAuth
func (h *MediaMetaApiHandler) Patch(g *gin.Context) {
	id, ok := h.bindMediaID(g)
	if !ok {
		return
	}
	if !h.requireMedia(g, id) {
		return
	}
	var req MediaMetaPatchReqType
	if err := g.ShouldBindJSON(&req); err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	doc, err := h.meta.PatchByMediaFileID(g.Request.Context(), id, types.MediaExtendedMetaPatch{
		LastPlayedAt: req.LastPlayedAt,
		Checkpoint:   req.Checkpoint,
		Score:        req.Score,
		Likes:        req.Likes,
		IsFavorite:   req.IsFavorite,
	})
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	g.JSON(http.StatusOK, doc)
}
func (h *MediaMetaApiHandler) AuthPatch() bool {
	return true
}
func (h *MediaMetaApiHandler) RelativePathPatch() string {
	return "/"
}

func (h *MediaMetaApiHandler) bindMediaID(g *gin.Context) (bson.ObjectID, bool) {
	var req MediaMetaIDReqType
	if err := g.ShouldBindUri(&req); err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return bson.ObjectID{}, false
	}
	id, err := bson.ObjectIDFromHex(req.ID)
	if err != nil {
		g.Error(wtypes.NewHttpError(fmt.Errorf("invalid id: %w", err), http.StatusBadRequest)) //nolint:golint,errcheck
		return bson.ObjectID{}, false
	}
	return id, true
}

func (h *MediaMetaApiHandler) requireMedia(g *gin.Context, id bson.ObjectID) bool {
	_, err := h.media.FindByID(g.Request.Context(), id)
	if err == nil {
		return true
	}
	if errors.Is(err, repository.ErrNotFound) {
		g.Error(wtypes.NewHttpError(err, http.StatusNotFound)) //nolint:golint,errcheck
		return false
	}
	g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
	return false
}
