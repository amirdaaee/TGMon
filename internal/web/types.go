package web

import "github.com/amirdaaee/TGMon/internal/types"

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
type MediaListReqType struct {
	Page int `form:"page"`
}
type MediaDelReqType struct {
	ID string `uri:"id" binding:"required"`
}
type MediaListResType []*types.MediaFileDoc

// ===
type JobReqDelReqType struct {
	ID string `uri:"id" binding:"required"`
}
type JobReqListResType []*types.JobReqDoc

// ===
type JobResPostReqType *types.JobResDoc
type JobResPostResType *types.JobResDoc

// ===
type InfoGetResType struct {
	MediaCount int64
}

// ===
type LoginPostReqType struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
type LoginPostResType struct {
	Token string `json:"token"`
}
