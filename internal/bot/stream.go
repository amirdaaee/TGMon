package bot

import (
	"context"
	"fmt"
	"io"

	"github.com/celestix/gotgproto"
	"github.com/gotd/td/tg"
	"github.com/sirupsen/logrus"
)

// ...
type TGReader struct {
	ctx           context.Context
	client        *gotgproto.Client
	location      *tg.InputDocumentFileLocation
	start         int64
	end           int64
	next          func() ([]byte, error)
	buffer        []byte
	bytesread     int64
	chunkSize     int64
	i             int64
	contentLength int64
}

func (*TGReader) Close() error {
	return nil
}

func (r *TGReader) Read(p []byte) (n int, err error) {

	if r.bytesread == r.contentLength {
		return 0, io.EOF
	}

	if r.i >= int64(len(r.buffer)) {
		r.buffer, err = r.next()
		if err != nil {
			logrus.WithError(err).Error("error getting next data")
			return 0, err
		}
		if len(r.buffer) == 0 {
			r.next = r.partStreamerFactory()
			r.buffer, err = r.next()
			if err != nil {
				return 0, err
			}
		}
		r.i = 0
	}
	n = copy(p, r.buffer[r.i:])
	r.i += int64(n)
	r.bytesread += int64(n)
	return n, nil
}

func (r *TGReader) partStreamerFactory() func() ([]byte, error) {
	start := r.start
	end := r.end
	offset := start - (start % r.chunkSize)

	firstPartCut := start - offset
	lastPartCut := (end % r.chunkSize) + 1
	partCount := int((end - offset + r.chunkSize) / r.chunkSize)
	currentPart := 1

	readData := func() ([]byte, error) {
		if currentPart > partCount {
			return make([]byte, 0), nil
		}
		res, err := r.getChunk(offset, r.chunkSize)
		if err != nil {
			return nil, err
		}
		if len(res) == 0 {
			return res, nil
		} else if partCount == 1 {
			res = res[firstPartCut:lastPartCut]
		} else if currentPart == 1 {
			res = res[firstPartCut:]
		} else if currentPart == partCount {
			res = res[:lastPartCut]
		}

		currentPart++
		offset += r.chunkSize
		return res, nil
	}
	return readData
}

func (r *TGReader) getChunk(offset int64, limit int64) ([]byte, error) {
	req := &tg.UploadGetFileRequest{
		Offset:   offset,
		Limit:    int(limit),
		Location: r.location,
	}
	logrus.WithField("offset", offset).WithField("limit", limit).Debug("serving chunk")
	res, err := r.client.API().UploadGetFile(r.ctx, req)

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
func NewTelegramReader(
	ctx context.Context,
	client *gotgproto.Client,
	location *tg.InputDocumentFileLocation,
	start int64,
	end int64,
	contentLength int64,
	chunkSize int64,
) (io.ReadCloser, error) {

	r := &TGReader{
		ctx:           ctx,
		location:      location,
		client:        client,
		start:         start,
		end:           end,
		chunkSize:     chunkSize,
		contentLength: contentLength,
	}
	r.next = r.partStreamerFactory()
	return r, nil
}
