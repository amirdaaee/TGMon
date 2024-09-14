package bot

import (
	"bytes"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/gotd/td/tg"
	"github.com/sirupsen/logrus"
)

// ...
type readMeta struct {
	doc           *Document
	start         int64
	end           int64
	chunkSize     int64
	i             int64
	contentLength int64
}
type TGReader struct {
	ctx      *gin.Context
	worker   *Worker
	rMeta    *readMeta
	DataChan chan []byte
	ll       *logrus.Entry
	buffer   *bytes.Buffer
}

func (r *TGReader) Close() error {
	r.ll.Debug("closed!")
	return nil
}
func (r *TGReader) StartReading() {

	ll := r.ll.WithField("func", "tgReader")
	ctx := r.ctx
	for {
		select {
		case <-ctx.Request.Context().Done():
			ll.Debug("request ctx done")
			r.finilazeStream()
			return
		default:
			offset, limit := r.getBound()
			ll2 := ll.WithField("offset", offset).WithField("limit", limit)
			if limit == 0 {
				ll2.Debug("end of file reached")
				r.finilazeStream()
				return
			}
			data, err := r.readMedia(offset, limit, r.rMeta.doc.AsInputDocumentFileLocation())
			if err != nil {
				if ctx.Err() == nil {
					ll2.WithError(err).Error("error reading media from tg")
				}
				r.finilazeStream()
				return
			}
			ll2.WithField("size", len(data)).Debug("read success")
			data = r.stripData(data, offset, limit)
			ll2.WithField("size", len(data)).Trace("stripped")
			r.DataChan <- data
			r.rMeta.i++
		}
	}
}
func (r *TGReader) Read(p []byte) (int, error) {
	ll := r.ll.WithField("func", "reader")
	if r.buffer.Len() == 0 {
		ll.Debug("waiting for data channel")
		data := <-r.DataChan
		if data == nil {
			ll.Debug("nil data received (EOF)")
			return 0, io.EOF
		}
		n, err := r.buffer.Write(data)
		if err != nil {
			return 0, fmt.Errorf("error appending buffer: %s", err)
		}
		ll.Debugf("wrote %d bytes to buffer", n)
	}
	n, err := r.buffer.Read(p)
	if err != nil {
		return 0, fmt.Errorf("error reading buffer: %s", err)
	}
	ll.Tracef("read %d bytes from buffer", n)
	return n, nil
}
func (r *TGReader) readMedia(offset int64, limit int, loc *tg.InputDocumentFileLocation) ([]byte, error) {
	req := &tg.UploadGetFileRequest{
		Offset:   offset,
		Limit:    limit,
		Location: loc,
	}
	res, err := r.worker.Client.API().UploadGetFile(r.ctx, req)
	if err != nil {
		return nil, err
	}
	switch result := res.(type) {
	case *tg.UploadFile:
		return result.Bytes, nil
	default:
		return nil, fmt.Errorf("unexpected type %T", r)
	}
}
func (r *TGReader) getBound() (int64, int) {
	startChunk := r.rMeta.start - (r.rMeta.start % r.rMeta.chunkSize)
	offsetByte := startChunk + r.rMeta.i*r.rMeta.chunkSize
	limit := r.rMeta.chunkSize
	if offsetByte > r.rMeta.start+r.rMeta.contentLength {
		limit = 0 //to signal for EOF
	}
	return offsetByte, int(limit)
}
func (r *TGReader) stripData(p []byte, offset int64, limit int) []byte {
	startI := int64(0)
	endI := int64(limit) - 1
	if offset < r.rMeta.start {
		startI = r.rMeta.start - offset
	}
	if offset+int64(limit) > r.rMeta.end {
		endI = r.rMeta.end - offset
	}

	return p[startI : endI+1]
}
func NewTelegramReader(
	ctx *gin.Context,
	worker *Worker,
	document *Document,
	start int64,
	end int64,
	contentLength int64,
	chunkSize int64,
) (*TGReader, error) {
	if err := worker.UpdateDocAccHash(document, ctx); err != nil {
		return nil, fmt.Errorf("can not update access hash: %s", err)
	}
	dChan := make(chan []byte, 4)
	r := &TGReader{
		ctx:      ctx,
		worker:   worker,
		ll:       logrus.WithField("doc id", document.ID),
		DataChan: dChan,
		buffer:   bytes.NewBuffer([]byte{}),
		rMeta: &readMeta{
			doc:           document,
			start:         start,
			end:           end,
			chunkSize:     chunkSize,
			contentLength: contentLength,
		},
	}
	return r, nil
}
func (r *TGReader) finilazeStream() {
	close(r.DataChan)
}
