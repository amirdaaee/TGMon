package crd

import (
	"context"
	"fmt"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2/bsonx"
	"github.com/chenmingyong0423/go-mongox/v2/builder/query"
	"github.com/chenmingyong0423/go-mongox/v2/finder"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

// MediaHandler implements IHandler for media resources.
type MediaHandler struct {
	dBContainer db.IDbContainer
	ll          *zap.Logger
}

var _ IReadHandler[types.MediaFileDoc] = (*MediaHandler)(nil)
var _ IListHandler[types.MediaFileDoc] = (*MediaHandler)(nil)
var _ IDeleteHandler[types.MediaFileDoc] = (*MediaHandler)(nil)

// =====
// @Summary	Read media
// @Tags		media
// @Produce	json
// @Param		id	path	string	true	"Media ID"
// @Success	200	{object}	MediaReadResType
// @Router		/api/media/{id}/ [get]
// @Security	ApiKeyAuth
func (h *MediaHandler) BindReadRequest(g *gin.Context) (bson.D, error) {
	var qID MediaReadReqType
	if err := g.ShouldBindUri(&qID); err != nil {
		return nil, err
	}
	idObj, err := bson.ObjectIDFromHex(qID.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}
	q := query.Id(idObj)
	return q, nil
}
func (h *MediaHandler) MarshalReadResponse(g *gin.Context, v *types.MediaFileDoc) (any, error) {
	ll := h.ll.Named("MarshalReadResponse")
	prevDocID, err := h.getNeighborsId(g.Request.Context(), v, query.NewBuilder().Lt, -1)
	if err != nil {
		ll.With(zap.Error(err)).Error("error finding previous document")
	}
	nextDocID, err := h.getNeighborsId(g.Request.Context(), v, query.NewBuilder().Gt, 1)
	if err != nil {
		ll.With(zap.Error(err)).Error("error finding previous document")
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
func (h *MediaHandler) BindListRequest(g *gin.Context, fnd finder.IFinder[types.MediaFileDoc]) (finder.IFinder[types.MediaFileDoc], error) {
	var v MediaListReqType
	const resultPerPage = 12
	if err := g.ShouldBindQuery(&v); err != nil {
		return nil, err
	}
	fnd = fnd.Sort(bson.D{{Key: "created_at", Value: -1}}).Skip(resultPerPage * int64(v.Page)).Limit(resultPerPage)
	return fnd, nil
}

// @Summary	Delete media
// @Tags		media
// @Produce	json
// @Param		id	path	string	true	"Media ID"
// @Success	200
// @Router		/api/media/{id}/ [delete]
// @Security	ApiKeyAuth
func (h *MediaHandler) BindDeleteRequest(g *gin.Context) (bson.D, error) {
	var qID MediaDelReqType
	if err := g.ShouldBindUri(&qID); err != nil {
		return nil, err
	}
	idObj, err := bson.ObjectIDFromHex(qID.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}
	q := query.Id(idObj)
	return q, nil
}
func (h *MediaHandler) MarshalListResponse(g *gin.Context, v []*types.MediaFileDoc) (any, error) {
	res := make([]*types.MediaFileDoc, len(v))
	for i, doc := range v {
		_v := types.MediaFileDoc(*doc)
		res[i] = &_v
	}
	total, err := h.dBContainer.GetMongoContainer().GetMediaFileCollection().Finder().Count(g.Request.Context())
	if err != nil {
		return nil, fmt.Errorf("error counting media: %w", err)
	}
	return MediaListResType{
		Media: res,
		Total: total,
	}, nil
}

func (h *MediaHandler) getNeighborsId(ctx context.Context, v *types.MediaFileDoc, qFactory func(string, any) *query.Builder, sort int) (*bson.ObjectID, error) {
	ll := h.ll.Named("getNeighborsId")
	fnd := h.dBContainer.GetMongoContainer().GetMediaFileCollection().Finder()
	createdAtField := "created_at"
	filter := qFactory(createdAtField, v.CreatedAt).Build()
	srt := bsonx.NewD().Add(createdAtField, sort).Build()
	ll.Sugar().Debugf("getting neighbors filter: %+v - sort: %+v", filter, srt)
	doc, err := fnd.Filter(filter).Sort(srt).FindOne(ctx)
	if err != nil {
		return nil, fmt.Errorf("error finding document: %w", err)
	}
	return &doc.ID, nil
}

func NewMediaHandler(dBContainer db.IDbContainer) *MediaHandler {
	return &MediaHandler{
		dBContainer: dBContainer,
		ll:          log.Named(log.WebModule, "MediaHandler"),
	}
}
