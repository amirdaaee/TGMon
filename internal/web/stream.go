package web

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/gin-gonic/gin"
	range_parser "github.com/quantumsheep/range-parser"
)

type mediaMetaData struct {
	start         int64
	end           int64
	contentLength int64
	mimeType      string
	fileSize      int64
	filename      string
}

func steam(ctx *gin.Context, mediaReq streamReq, wp *bot.WorkerPool, mongo *db.Mongo, chunckSize int64, profileFile string) error {
	r := ctx.Request
	mediaID := mediaReq.ID
	var medDoc db.MediaFileDoc
	// ...
	if err := mongo.DocGetById(ctx, mediaID, &medDoc, nil); err != nil {
		return fmt.Errorf("error DocGetById: %s", err)
	}
	metaData, err := getMetaData(ctx, medDoc)
	if err != nil {
		return fmt.Errorf("error getMetaData: %s", err)
	}
	status, headers := getStreamHeaders(ctx, metaData)
	//...
	if r.Method == "HEAD" {
		ctx.Writer.WriteHeader(status)
		for k, v := range headers {
			ctx.Header(k, v)
		}
		return nil
	}

	worker := wp.SelectNextWorker()
	docMsg, err := worker.GetChannelMessages(ctx, []int{medDoc.MessageID})
	if err != nil {
		return fmt.Errorf("error GetMessages: %s", err)
	}
	doc := bot.TelegramDocument{}
	doc.FromMessage(docMsg.Messages[0])
	lr, err := bot.NewTelegramReader(ctx, worker, &doc, metaData.start, metaData.end, metaData.contentLength, chunckSize, profileFile)
	if err != nil {
		return fmt.Errorf("error NewTelegramReader: %s", err)

	}
	go lr.StartReading()
	// DONT return error after this
	ctx.DataFromReader(status, metaData.contentLength, metaData.mimeType, lr, headers)
	return nil
}
func getStreamHeaders(ctx *gin.Context, meta *mediaMetaData) (int, map[string]string) {
	r := ctx.Request
	rangeHeader := r.Header.Get("Range")
	head := map[string]string{}
	var status int
	if rangeHeader == "" {
		status = http.StatusOK
	} else {
		status = http.StatusPartialContent
		head["Content-Range"] = fmt.Sprintf("bytes %d-%d/%d", meta.start, meta.end, meta.fileSize)
	}
	disposition := "inline"
	if ctx.Query("d") == "true" {
		disposition = "attachment"
	}
	head["Content-Disposition"] = fmt.Sprintf("%s; filename=\"%s\"", disposition, meta.filename)
	head["Content-Type"] = meta.mimeType
	head["Content-Length"] = strconv.FormatInt(meta.contentLength, 10)

	return status, head
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
