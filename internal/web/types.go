package web

import "github.com/amirdaaee/TGMon/internal/db"

type streamReq struct {
	ID string `uri:"mediaID" binding:"required"`
}
type mediaListReq struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

type mediaListRes struct {
	Media []db.MediaFileDoc
	Total int64
}
type loginReq struct {
	Username string
	Password string
}