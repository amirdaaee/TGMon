package web

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/gin-gonic/gin"
	range_parser "github.com/quantumsheep/range-parser"
	"github.com/sirupsen/logrus"
)

type mediaMetaData struct {
	start         int64
	end           int64
	contentLength int64
	mimeType      string
	fileSize      int64
	filename      string
}

func steam(ctx *gin.Context, mediaReq streamReq, wp *bot.WorkerPool, mongo *db.Mongo, chunckSize int64) error {
	w := ctx.Writer
	r := ctx.Request
	mediaID := mediaReq.ID
	var med db.MediaFileDoc
	if err := mongo.DocGetById(ctx, mediaID, &med, nil); err != nil {
		return err
	}
	metaData, err := getMetaData(ctx, med)
	if err != nil {
		return err
	}
	if err := writeStramHeaders(ctx, metaData); err != nil {
		return err
	}
	//...
	worker := wp.GetNextWorker()
	if r.Method != "HEAD" {
		lr, _ := bot.NewTelegramReader(ctx, worker.Client, med.Location, metaData.start, metaData.end, metaData.contentLength, chunckSize)
		written, err := io.CopyN(w, lr, metaData.contentLength)
		if err != nil {
			logrus.WithError(err).Errorf("error streaming after %d", written)
			return err
		}
	}
	return nil
}
func writeStramHeaders(ctx *gin.Context, meta *mediaMetaData) error {
	r := ctx.Request
	w := ctx.Writer
	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		w.WriteHeader(http.StatusOK)
	} else {
		ctx.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", meta.start, meta.end, meta.fileSize))
		w.WriteHeader(http.StatusPartialContent)
	}
	ctx.Header("Content-Type", meta.mimeType)
	ctx.Header("Content-Length", strconv.FormatInt(meta.contentLength, 10))
	disposition := "inline"
	if ctx.Query("d") == "true" {
		disposition = "attachment"
	}
	ctx.Header("Content-Disposition", fmt.Sprintf("%s; filename=\"%s\"", disposition, meta.filename))
	return nil
}
func getMetaData(ctx *gin.Context, media db.MediaFileDoc) (*mediaMetaData, error) {
	r := ctx.Request

	var start, end int64
	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		start = 0
		end = media.FileSize - 1
	} else {
		ranges, err := range_parser.Parse(media.FileSize, r.Header.Get("Range"))
		if err != nil {
			return nil, err
		}
		start = ranges[0].Start
		end = ranges[0].End
		ctx.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, media.FileSize))
	}
	contentLength := end - start + 1
	metaData := mediaMetaData{
		start:         start,
		end:           end,
		contentLength: contentLength,
		mimeType:      media.MimeType,
		fileSize:      media.FileSize,
		filename:      media.FileName,
	}
	if metaData.mimeType == "" {
		metaData.mimeType = "application/octet-stream"
	}
	return &metaData, nil
}
