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

type IHandler[T any] interface {
	BindCreateRequest(g *gin.Context) (*T, error)
	BindListRequest(g *gin.Context, fnd finder.IFinder[T]) (finder.IFinder[T], error)
	BindDeleteRequest(g *gin.Context) (bson.D, error)
	MarshalCreateResponse(*T) any
	MarshalListResponse([]*T) any
}

// IHandler defines methods for binding requests and marshaling responses for a resource type T.

// MediaHandler implements IHandler for media resources.
type MediaHandler struct{}

// JobReqHandler implements IHandler for media resources.
type JobReqHandler struct{}

// JobResHandler implements IHandler for media resources.
type JobResHandler struct{}

var _ IHandler[types.MediaFileDoc] = (*MediaHandler)(nil)
var _ IHandler[types.JobReqDoc] = (*JobReqHandler)(nil)
var _ IHandler[types.JobResDoc] = (*JobResHandler)(nil)

// =====
func (h *MediaHandler) BindCreateRequest(g *gin.Context) (*types.MediaFileDoc, error) {
	return nil, fmt.Errorf("not supported method")
}
func (h *MediaHandler) BindListRequest(g *gin.Context, fnd finder.IFinder[types.MediaFileDoc]) (finder.IFinder[types.MediaFileDoc], error) {
	var v MediaListReqType
	const resultPerPage = 12
	if err := g.ShouldBindQuery(&v); err != nil {
		return nil, err
	}
	fnd = fnd.Sort(bson.D{{Key: "created_at", Value: -1}}).Skip(resultPerPage * int64(v.Page)).Limit(resultPerPage)
	return fnd, nil
}
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
func (h *MediaHandler) MarshalCreateResponse(v *types.MediaFileDoc) any {
	return nil
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
func (h *JobReqHandler) BindCreateRequest(g *gin.Context) (*types.JobReqDoc, error) {
	return nil, fmt.Errorf("not supported method")
}
func (h *JobReqHandler) BindListRequest(g *gin.Context, fnd finder.IFinder[types.JobReqDoc]) (finder.IFinder[types.JobReqDoc], error) {
	return fnd, nil
}
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
func (h *JobReqHandler) MarshalCreateResponse(v *types.JobReqDoc) any {
	return nil
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
func (h *JobResHandler) BindCreateRequest(g *gin.Context) (*types.JobResDoc, error) {
	var v JobResPostReqType
	if err := g.ShouldBindJSON(&v); err != nil {
		return nil, err
	}
	return v, nil
}
func (h *JobResHandler) BindListRequest(g *gin.Context, fnd finder.IFinder[types.JobResDoc]) (finder.IFinder[types.JobResDoc], error) {
	return nil, fmt.Errorf("not supported method")
}
func (h *JobResHandler) BindDeleteRequest(g *gin.Context) (bson.D, error) {
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
func (h *JobResHandler) MarshalCreateResponse(v *types.JobResDoc) any {
	return nil
}
func (h *JobResHandler) MarshalListResponse(v []*types.JobResDoc) any {
	res := make([]*types.JobResDoc, len(v))
	for i, doc := range v {
		_v := types.JobResDoc(*doc)
		res[i] = &_v
	}
	return JobResListReqType(res)
}
