package crd

import (
	"fmt"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/repository"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

// MediaHandler implements IHandler for media resources.
type MediaHandler struct {
	media repository.MediaFileRepository
	ll    *zap.Logger
}

var _ IReadHandler[types.MediaFileDoc] = (*MediaHandler)(nil)
var _ IListHandler = (*MediaHandler)(nil)
var _ IDeleteHandler = (*MediaHandler)(nil)

// =====
// @Summary	Read media
// @Tags		media
// @Produce	json
// @Param		id	path	string	true	"Media ID"
// @Success	200	{object}	MediaReadResType
// @Router		/api/media/{id}/ [get]
// @Security	ApiKeyAuth
func (h *MediaHandler) BindReadRequest(g *gin.Context) (bson.ObjectID, error) {
	var qID MediaReadReqType
	if err := g.ShouldBindUri(&qID); err != nil {
		return bson.ObjectID{}, err
	}
	idObj, err := bson.ObjectIDFromHex(qID.ID)
	if err != nil {
		return bson.ObjectID{}, fmt.Errorf("invalid id: %w", err)
	}
	return idObj, nil
}
func (h *MediaHandler) MarshalReadResponse(g *gin.Context, v *types.MediaFileDoc) (any, error) {
	ll := h.ll.Named("MarshalReadResponse")
	prevDocID, nextDocID, err := h.media.FindNeighbors(g.Request.Context(), v)
	if err != nil {
		ll.With(zap.Error(err)).Error("error finding neighbor documents")
	}
	return MediaReadResType{
		Media:  v,
		PervID: prevDocID,
		NextID: nextDocID,
	}, nil
}

// @Summary	List media
// @Tags		media
// @Produce	json
// @Param		page	query	int	false	"page"
// @Success	200		{object}	MediaListResType
// @Router		/api/media/ [get]
// @Security	ApiKeyAuth
func (h *MediaHandler) HandleList(g *gin.Context) (any, error) {
	var v MediaListReqType
	const resultPerPage = 12
	if err := g.ShouldBindQuery(&v); err != nil {
		return nil, err
	}
	docs, err := h.media.ListPage(g.Request.Context(), v.Page, resultPerPage)
	if err != nil {
		return nil, fmt.Errorf("error listing media: %w", err)
	}
	res := make([]*types.MediaFileDoc, len(docs))
	for i, doc := range docs {
		_v := types.MediaFileDoc(*doc)
		res[i] = &_v
	}
	total, err := h.media.Count(g.Request.Context())
	if err != nil {
		return nil, fmt.Errorf("error counting media: %w", err)
	}
	return MediaListResType{
		Media: res,
		Total: total,
	}, nil
}

// @Summary	Delete media
// @Tags		media
// @Produce	json
// @Param		id	path	string	true	"Media ID"
// @Success	200
// @Router		/api/media/{id}/ [delete]
// @Security	ApiKeyAuth
func (h *MediaHandler) BindDeleteRequest(g *gin.Context) (bson.ObjectID, error) {
	var qID MediaDelReqType
	if err := g.ShouldBindUri(&qID); err != nil {
		return bson.ObjectID{}, err
	}
	idObj, err := bson.ObjectIDFromHex(qID.ID)
	if err != nil {
		return bson.ObjectID{}, fmt.Errorf("invalid id: %w", err)
	}
	return idObj, nil
}

func NewMediaHandler(media repository.MediaFileRepository) *MediaHandler {
	return &MediaHandler{
		media: media,
		ll:    log.Named(log.WebModule, "MediaHandler"),
	}
}
