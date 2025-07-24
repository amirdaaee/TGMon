package web

import (
	"github.com/amirdaaee/TGMon/internal/types"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ApiType string

const (
	No     ApiType = "No"
	NoAuth ApiType = "NO_AUTH"
	Auth   ApiType = "AUTH"
)

type StreamMetaData struct {
	Start         int64
	End           int64
	ContentLength int64
	MimeType      string
	FileSize      int64
	Filename      string
}

type StreamReq struct {
	ID string `uri:"mediaID" binding:"required"`
}

// ===
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

// ===
type InfoGetResType struct {
	MediaCount int64
}

// ===
type LoginPostReqType struct {
	Username string `binding:"required"`
	Password string `binding:"required"`
}
type LoginPostResType struct {
	Token string
}

// ===
type RandomMediaGetResType struct {
	MediaID *bson.ObjectID
}
