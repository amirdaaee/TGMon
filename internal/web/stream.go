package web

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
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
	streamPool  stream.IWorkerPool
}

func (s *Streamhandler) Stream(g *gin.Context) {
	r := g.Request
	var req StreamReq
	if err := g.ShouldBindUri(&req); err != nil {
		g.Error(NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	media, err := s.getMedia(g, req.ID)
	if err != nil {
		g.Error(err) //nolint:golint,errcheck
		return
	}
	meta, err := s.getStreamMetaData(r, *media)
	if err != nil {
		g.Error(NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	status, headers := s.getStreamHeaders(r, meta, g.Query("d") == "true")
	if r.Method == "HEAD" {
		g.Writer.WriteHeader(status)
		for k, v := range headers {
			g.Header(k, v)
		}
		return
	}
	streamer, err := s.streamPool.Stream(g.Request.Context(), media.MessageID, meta.Start, meta.End)
	if err != nil {
		g.Error(NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	defer runtime.GC()
	// remove content-length from header map
	delete(headers, "Content-Length")
	delete(headers, "Content-Type")
	g.DataFromReader(status, meta.ContentLength, meta.MimeType, streamer.GetBuffer(), headers)
}
func (s *Streamhandler) getMedia(g *gin.Context, id string) (*types.MediaFileDoc, error) {
	if id == "" {
		return nil, NewHttpError(errors.New("mediaID is required"), http.StatusBadRequest)
	}
	idObj, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, NewHttpError(fmt.Errorf("error parsing mediaID: %w", err), http.StatusBadRequest)
	}
	media, err := s.mediaFacade.GetCollection().Finder().Filter(query.Id(idObj)).Find(g.Request.Context())
	if err != nil {
		return nil, NewHttpError(err, http.StatusInternalServerError)
	}
	if len(media) == 0 {
		return nil, NewHttpError(fmt.Errorf("media (%s) not found", id), http.StatusNotFound)
	}
	return media[0], nil
}
func (s *Streamhandler) getStreamMetaData(req *http.Request, media types.MediaFileDoc) (*StreamMetaData, error) {
	ll := s.getLogger("getStreamMetaData")
	var start, end int64
	rangeHeader := req.Header.Get("Range")
	fileSize := media.Meta.FileSize
	if rangeHeader == "" {
		ll.Debug("no range header")
		start = 0
		end = fileSize - 1
	} else {
		ll.Debugf("range header %s", rangeHeader)
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
	if metaData.Filename == "" {
		metaData.Filename = fmt.Sprintf("%d.mp4", media.Meta.FileID)
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
func NewStreamHandler(dbContainer db.IDbContainer, mediaFacade facade.IFacade[types.MediaFileDoc], wp stream.IWorkerPool) *Streamhandler {
	return &Streamhandler{
		dbContainer: dbContainer,
		mediaFacade: mediaFacade,
		streamPool:  wp,
	}
}
