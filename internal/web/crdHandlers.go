package web

import (
	"fmt"

	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2/builder/query"
	"github.com/chenmingyong0423/go-mongox/v2/finder"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Package api provides handler interfaces and implementations for API resource operations.
type ICreateApiHandler[T any] interface {
	BindCreateRequest(g *gin.Context) (*T, error)
	MarshalCreateResponse(*T) any
}
type IListApiHandler[T any] interface {
	BindListRequest(g *gin.Context, fnd finder.IFinder[T]) (finder.IFinder[T], error)
	MarshalListResponse([]*T) any
}
type IDeleteApiHandler[T any] interface {
	BindDeleteRequest(g *gin.Context) (bson.D, error)
}

// IHandler defines methods for binding requests and marshaling responses for a resource type T.

// MediaHandler implements IHandler for media resources.
type MediaHandler struct{}

// JobReqHandler implements IHandler for media resources.
type JobReqHandler struct{}

// JobResHandler implements IHandler for media resources.
type JobResHandler struct{}

var _ IListApiHandler[types.MediaFileDoc] = (*MediaHandler)(nil)
var _ IDeleteApiHandler[types.MediaFileDoc] = (*MediaHandler)(nil)
var _ IListApiHandler[types.JobReqDoc] = (*JobReqHandler)(nil)
var _ IDeleteApiHandler[types.JobReqDoc] = (*JobReqHandler)(nil)
var _ ICreateApiHandler[types.JobResDoc] = (*JobResHandler)(nil)

// =====
// @Summary	List media
// @Tags		media
// @Produce	json
// @Param		page	query	int	false	"page"
// @Success	200		{array}	types.MediaFileDoc
// @Router		/api/media [get]
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
// @Router		/api/media/{id} [delete]
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
func (h *MediaHandler) MarshalListResponse(v []*types.MediaFileDoc) any {
	res := make([]*types.MediaFileDoc, len(v))
	for i, doc := range v {
		_v := types.MediaFileDoc(*doc)
		res[i] = &_v
	}
	return MediaListResType(res)
}

// =====
// @Summary	List job requests
// @Tags		jobReq
// @Produce	json
// @Success	200	{array}	types.JobReqDoc
// @Router		/api/jobReq [get]
// @Security	ApiKeyAuth
func (h *JobReqHandler) BindListRequest(g *gin.Context, fnd finder.IFinder[types.JobReqDoc]) (finder.IFinder[types.JobReqDoc], error) {
	return fnd, nil
}

// @Summary	Delete job request
// @Tags		jobReq
// @Produce	json
// @Param		id	path		string	true	"Job Request ID"
// @Success	200	{string}	string	"OK"
// @Router		/api/jobReq/{id} [delete]
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
func (h *JobReqHandler) MarshalListResponse(v []*types.JobReqDoc) any {
	res := make([]*types.JobReqDoc, len(v))
	for i, doc := range v {
		_v := types.JobReqDoc(*doc)
		res[i] = &_v
	}
	return JobReqListResType(res)
}

// =====
//
//	@Summary	Create job response
//	@Tags		jobRes
//	@Accept		json
//	@Produce	json
//	@Param		data	body		JobResPostReqType	true	"Job Response Data"
//	@Success	200		{object}	types.JobResDoc
//	@Router		/api/jobRes [post]
//	@Security	ApiKeyAuth
func (h *JobResHandler) BindCreateRequest(g *gin.Context) (*types.JobResDoc, error) {
	var v JobResPostReqType
	if err := g.ShouldBindJSON(&v); err != nil {
		return nil, err
	}
	return v, nil
}
func (h *JobResHandler) MarshalCreateResponse(v *types.JobResDoc) any {
	return v
}
