package web

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
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Package api provides handler interfaces and implementations for API resource operations.
type ICreateApiHandler[T any] interface {
	BindCreateRequest(g *gin.Context) (*T, error)
	MarshalCreateResponse(g *gin.Context, v *T) (any, error)
}
type IReadApiHandler[T any] interface {
	BindReadRequest(g *gin.Context) (bson.D, error)
	MarshalReadResponse(g *gin.Context, v *T) (any, error)
}
type IListApiHandler[T any] interface {
	BindListRequest(g *gin.Context, fnd finder.IFinder[T]) (finder.IFinder[T], error)
	MarshalListResponse(g *gin.Context, v []*T) (any, error)
}
type IDeleteApiHandler[T any] interface {
	BindDeleteRequest(g *gin.Context) (bson.D, error)
}

// IHandler defines methods for binding requests and marshaling responses for a resource type T.

// MediaHandler implements IHandler for media resources.
type MediaHandler struct {
	DBContainer db.IDbContainer
}

// JobReqHandler implements IHandler for media resources.
type JobReqHandler struct{}

// JobResHandler implements IHandler for media resources.
type JobResHandler struct{}

var _ IReadApiHandler[types.MediaFileDoc] = (*MediaHandler)(nil)
var _ IListApiHandler[types.MediaFileDoc] = (*MediaHandler)(nil)
var _ IDeleteApiHandler[types.MediaFileDoc] = (*MediaHandler)(nil)

var _ ICreateApiHandler[types.JobReqDoc] = (*JobReqHandler)(nil)
var _ IListApiHandler[types.JobReqDoc] = (*JobReqHandler)(nil)
var _ IDeleteApiHandler[types.JobReqDoc] = (*JobReqHandler)(nil)

var _ ICreateApiHandler[types.JobResDoc] = (*JobResHandler)(nil)

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
	ll := h.getLogger("MarshalReadResponse")
	prevDocID, err := h.getNeighborsId(g.Request.Context(), v, query.NewBuilder().Lt, -1)
	if err != nil {
		ll.WithError(err).Error("error finding previous document")
	}
	nextDocID, err := h.getNeighborsId(g.Request.Context(), v, query.NewBuilder().Gt, 1)
	if err != nil {
		ll.WithError(err).Error("error finding previous document")
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
	total, err := h.DBContainer.GetMongoContainer().GetMediaFileCollection().Finder().Count(g.Request.Context())
	if err != nil {
		return nil, fmt.Errorf("error counting media: %w", err)
	}
	return MediaListResType{
		Media: res,
		Total: total,
	}, nil
}
func (h *MediaHandler) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.WebModule).WithField("func", fmt.Sprintf("%T.%s", h, fn))
}
func (h *MediaHandler) getNeighborsId(ctx context.Context, v *types.MediaFileDoc, qFactory func(string, any) *query.Builder, sort int) (*bson.ObjectID, error) {
	fnd := h.DBContainer.GetMongoContainer().GetMediaFileCollection().Finder()
	createdAtField := "created_at"
	filter := qFactory(createdAtField, v.CreatedAt).Build()
	srt := bsonx.NewD().Add(createdAtField, sort).Build()
	h.getLogger("getNeighbors").Infof("getting neighbors filter: %+v - sort: %+v", filter, srt)
	doc, err := fnd.Filter(filter).Sort(srt).FindOne(ctx)
	if err != nil {
		return nil, fmt.Errorf("error finding document: %w", err)
	}
	return &doc.ID, nil
}

// =====
// @Summary	Create job request
// @Tags		jobReq
// @Produce	json
// @Param		data	body		types.JobReqDoc	true	"Job Request Data"
// @Success	200	{object}	types.JobReqDoc
// @Router		/api/jobReq/ [post]
// @Security	ApiKeyAuth
func (h *JobReqHandler) BindCreateRequest(g *gin.Context) (*types.JobReqDoc, error) {
	var v types.JobReqDoc
	if err := g.ShouldBindJSON(&v); err != nil {
		return nil, err
	}
	return &v, nil
}
func (h *JobReqHandler) MarshalCreateResponse(g *gin.Context, v *types.JobReqDoc) (any, error) {
	return v, nil
}

// @Summary	List job requests
// @Tags		jobReq
// @Produce	json
// @Success	200	{array}	types.JobReqDoc
// @Router		/api/jobReq/ [get]
// @Security	ApiKeyAuth
func (h *JobReqHandler) BindListRequest(g *gin.Context, fnd finder.IFinder[types.JobReqDoc]) (finder.IFinder[types.JobReqDoc], error) {
	return fnd, nil
}

// @Summary	Delete job request
// @Tags		jobReq
// @Produce	json
// @Param		id	path		string	true	"Job Request ID"
// @Success	200	{string}	string	"OK"
// @Router		/api/jobReq/{id}/ [delete]
// @Security	ApiKeyAuth
func (h *JobReqHandler) BindDeleteRequest(g *gin.Context) (bson.D, error) {
	var qID JobReqDelReqType
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
func (h *JobReqHandler) MarshalListResponse(g *gin.Context, v []*types.JobReqDoc) (any, error) {
	res := make([]*types.JobReqDoc, len(v))
	for i, doc := range v {
		_v := types.JobReqDoc(*doc)
		res[i] = &_v
	}
	return JobReqListResType(res), nil
}

// =====
//
//	@Summary	Create job response
//	@Tags		jobRes
//	@Accept		json
//	@Produce	json
//	@Param		data	body		types.JobResDoc	true	"Job Response Data"
//	@Success	200		{object}	types.JobResDoc
//	@Router		/api/jobRes/ [post]
//	@Security	ApiKeyAuth
func (h *JobResHandler) BindCreateRequest(g *gin.Context) (*types.JobResDoc, error) {
	var v types.JobResDoc
	if err := g.ShouldBindJSON(&v); err != nil {
		return nil, err
	}
	return &v, nil
}
func (h *JobResHandler) MarshalCreateResponse(g *gin.Context, v *types.JobResDoc) (any, error) {
	return v, nil
}
