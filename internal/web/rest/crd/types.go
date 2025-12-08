package crd

import (
	"github.com/amirdaaee/TGMon/internal/types"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type MediaReadReqType struct {
	ID string `uri:"id" binding:"required"`
}
type MediaReadResType struct {
	Media  *types.MediaFileDoc
	PervID *bson.ObjectID `json:"pervID"`
	NextID *bson.ObjectID `json:"nextID"`
}
type MediaListReqType struct {
	Page int `form:"page"`
}
type MediaDelReqType struct {
	ID string `uri:"id" binding:"required"`
}
type MediaListResType struct {
	Media []*types.MediaFileDoc
	Total int64
}

// ===
type JobReqDelReqType struct {
	ID string `uri:"id" binding:"required"`
}
type JobReqListResType []*types.JobReqDoc
