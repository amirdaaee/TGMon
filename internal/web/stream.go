package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/facade"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/stream"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2/builder/query"
	"github.com/gin-gonic/gin"
	range_parser "github.com/quantumsheep/range-parser"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Streamhandler struct {
	dbContainer db.IDbContainer
	mediaFacade facade.IFacade[types.MediaFileDoc]
	streamPool  stream.IWorkerContainer
}

func (s *Streamhandler) Stream(g *gin.Context) {
	ll := s.getLogger("Stream")
	r := g.Request
	var req StreamReq
	if err := g.ShouldBindUri(&req); err != nil {
		g.AbortWithError(http.StatusBadRequest, err)
		return
	}
	media := s.getMedia(g, req.ID)
	if media == nil {
		// s.getMedia should have already aborted the request with error
		return
	}
	meta, err := s.getStreamMetaData(r, *media)
	if err != nil {
		g.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	status, headers := s.getStreamHeaders(r, meta, g.Query("d") == "true")
	g.Writer.WriteHeader(status)
	for k, v := range headers {
		g.Header(k, v)
	}
	if r.Method == "HEAD" {
		return
	}
	wp := s.streamPool.GetWorkerPool()
	if err := wp.Stream(g.Request.Context(), media.MessageID, meta.Start, g.Writer); err != nil {
		ll.WithError(err).Error("error streaming media")
	}
	ll.Debug("stream finished")
}
func (s *Streamhandler) getMedia(g *gin.Context, id string) *types.MediaFileDoc {
	if id == "" {
		g.AbortWithError(http.StatusBadRequest, errors.New("mediaID is required"))
		return nil
	}
	idObj, err := bson.ObjectIDFromHex(id)
	if err != nil {
		g.AbortWithError(http.StatusBadRequest, fmt.Errorf("error parsing mediaID: %w", err))
		return nil
	}
	media, err := s.mediaFacade.Read(g.Request.Context(), query.Id(idObj))
	if err != nil {
		g.AbortWithError(http.StatusInternalServerError, err)
		return nil
	}
	if len(media) == 0 {
		g.AbortWithError(http.StatusNotFound, fmt.Errorf("media (%s) not found", id))
		return nil
	}
	return media[0]
}
func (s *Streamhandler) getStreamMetaData(req *http.Request, media types.MediaFileDoc) (*StreamMetaData, error) {
	ll := s.getLogger("getStreamMetaData")
	var start, end int64
	rangeHeader := req.Header.Get("Range")
	fileSize := media.Meta.FileSize
	if rangeHeader == "" {
		start = 0
		end = fileSize - 1
	} else {
		ranges, err := range_parser.Parse(fileSize, rangeHeader)
		if err != nil {
			return nil, err
		}
		start = ranges[0].Start
		end = ranges[0].End
	}
	contentLength := end - start + 1
	metaData := StreamMetaData{
		Start:         start,
		End:           end,
		ContentLength: contentLength,
		MimeType:      media.Meta.MimeType,
		FileSize:      media.Meta.FileSize,
		Filename:      media.Meta.FileName,
	}

	if metaData.MimeType == "" {
		metaData.MimeType = "application/octet-stream"
	}
	ll.Debugf("meta data: %+v", metaData)
	return &metaData, nil
}
func (s *Streamhandler) getStreamHeaders(req *http.Request, meta *StreamMetaData, download bool) (int, map[string]string) {
	ll := s.getLogger("getStreamHeaders")
	rangeHeader := req.Header.Get("Range")
	head := map[string]string{}
	var status int
	if rangeHeader == "" {
		status = http.StatusOK
	} else {
		status = http.StatusPartialContent
		head["Content-Range"] = fmt.Sprintf("bytes %d-%d/%d", meta.Start, meta.End, meta.FileSize)
	}
	disposition := ""
	if download {
		disposition = "attachment"
	} else {
		disposition = "inline"
	}
	head["Content-Disposition"] = fmt.Sprintf("%s; filename=\"%s\"", disposition, meta.Filename)
	head["Content-Type"] = meta.MimeType
	head["Content-Length"] = strconv.FormatInt(meta.ContentLength, 10)
	ll.Debugf("stream response headers: %+v", head)
	return status, head
}
func (s *Streamhandler) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.WebModule).WithField("func", fmt.Sprintf("%T.%s", s, fn))
}
func NewStreamHandler(dbContainer db.IDbContainer, mediaFacade facade.IFacade[types.MediaFileDoc], wp stream.IWorkerContainer) *Streamhandler {
	return &Streamhandler{
		dbContainer: dbContainer,
		mediaFacade: mediaFacade,
		streamPool:  wp,
	}
}
