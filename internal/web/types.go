package web

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
