package stream

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strconv"

	ftypes "github.com/amirdaaee/TGMon/internal/facade/types"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/stream"
	"github.com/amirdaaee/TGMon/internal/types"
	wtypes "github.com/amirdaaee/TGMon/internal/web/types"
	"github.com/amirdaaee/TGMon/internal/worker"
	"github.com/gin-gonic/gin"
	range_parser "github.com/quantumsheep/range-parser"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

type Streamhandler struct {
	mediaFacade  ftypes.IFacade[types.MediaFileDoc]
	workerPool   worker.IWorkerPool
	streamConfig *stream.StreamConfig
	ll           *zap.Logger
}

var _ wtypes.Registereable = (*Streamhandler)(nil)

func (s *Streamhandler) Stream(g *gin.Context) {
	r := g.Request
	var req StreamReq
	if err := g.ShouldBindUri(&req); err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	media, err := s.getMedia(g, req.ID)
	if err != nil {
		g.Error(err) //nolint:golint,errcheck
		return
	}
	meta, err := s.getStreamMetaData(r, *media)
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
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
	worker := s.workerPool.GetNextWorker()
	if worker == nil {
		g.Error(wtypes.NewHttpError(errors.New("no available worker"), http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	streamer, err := worker.Stream(g.Request.Context(), media.MessageID, &stream.StreamOpts{Start: meta.Start, End: meta.End})
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	defer runtime.GC()
	defer streamer.Close()
	delete(headers, "Content-Length")
	delete(headers, "Content-Type")
	g.DataFromReader(status, meta.ContentLength, meta.MimeType, streamer, headers)
}
func (s *Streamhandler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) error {
	r.Match([]string{"HEAD", "GET"}, "/stream/:mediaID", s.Stream)
	return nil
}
func (s *Streamhandler) RegisterToRoot() bool {
	return true
}
func (s *Streamhandler) getMedia(g *gin.Context, id string) (*types.MediaFileDoc, error) {
	if id == "" {
		return nil, wtypes.NewHttpError(errors.New("mediaID is required"), http.StatusBadRequest)
	}
	idObj, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, wtypes.NewHttpError(fmt.Errorf("error parsing mediaID: %w", err), http.StatusBadRequest)
	}
	media, err := s.mediaFacade.FindByID(g.Request.Context(), idObj)
	if err != nil {
		if errors.Is(err, ftypes.ErrNoDocumentsFound) {
			return nil, wtypes.NewHttpError(fmt.Errorf("media (%s) not found", id), http.StatusNotFound)
		}
		return nil, wtypes.NewHttpError(err, http.StatusInternalServerError)
	}
	return media, nil
}
func (s *Streamhandler) getStreamMetaData(req *http.Request, media types.MediaFileDoc) (*StreamMetaData, error) {
	ll := s.ll.Named("getStreamMetaData")
	var start, end int64
	rangeHeader := req.Header.Get("Range")
	fileSize := media.Meta.FileSize
	if rangeHeader == "" {
		ll.Debug("no range header")
		start = 0
		end = fileSize - 1
	} else {
		ll.Sugar().Debugf("range header %s", rangeHeader)
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
		Filename:      media.NameExt(),
	}
	if metaData.MimeType == "" {
		metaData.MimeType = "application/octet-stream"
	}
	ll.Sugar().Debugf("meta data: %+v", metaData)
	return &metaData, nil
}
func (s *Streamhandler) getStreamHeaders(req *http.Request, meta *StreamMetaData, download bool) (int, map[string]string) {
	ll := s.ll.Named("getStreamHeaders")
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
	ll.Sugar().Debugf("stream response headers: %+v", head)
	return status, head
}
func NewStreamHandler(mediaFacade ftypes.IFacade[types.MediaFileDoc], wp worker.IWorkerPool, streamConfig *stream.StreamConfig) *Streamhandler {
	return &Streamhandler{
		mediaFacade:  mediaFacade,
		workerPool:   wp,
		streamConfig: streamConfig,
		ll:           log.Named(log.WebModule, "StreamHandler"),
	}
}
