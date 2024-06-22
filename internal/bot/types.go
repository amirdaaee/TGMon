package bot

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/celestix/gotgproto"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"
)

// ...
type Worker struct {
	ID     int
	Client *gotgproto.Client
	Self   *tg.User
}

func (w *Worker) String() string {
	return fmt.Sprintf("{Worker (%d|@%s)}", w.ID, w.Self.Username)
}

// ...
type BotWorkers struct {
	Bots     []*Worker
	starting int
	index    int
	mut      sync.Mutex
	log      *zap.Logger
}

func (w *BotWorkers) Init(log *zap.Logger) {
	w.log = log.Named("Workers")
}

func (w *BotWorkers) AddDefaultClient(client *gotgproto.Client, self *tg.User) {
	if w.Bots == nil {
		w.Bots = make([]*Worker, 0)
	}
	w.incStarting()
	w.Bots = append(w.Bots, &Worker{
		Client: client,
		ID:     w.starting,
		Self:   self,
	})
	w.log.Sugar().Info("Default bot loaded")
}

func (w *BotWorkers) incStarting() {
	w.mut.Lock()
	defer w.mut.Unlock()
	w.starting++
}

func (w *BotWorkers) Add(token string) (err error) {
	w.incStarting()
	var botID int = w.starting
	client, err := startClient(w.log, token, botID)
	if err != nil {
		return err
	}
	w.log.Sugar().Infof("Bot @%s loaded with ID %d", client.Self.Username, botID)
	w.Bots = append(w.Bots, &Worker{
		Client: client,
		ID:     botID,
		Self:   client.Self,
	})
	return nil
}

// ...
type telegramReader struct {
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

func (*telegramReader) Close() error {
	return nil
}

func (r *telegramReader) Read(p []byte) (n int, err error) {

	if r.bytesread == r.contentLength {
		return 0, io.EOF
	}

	if r.i >= int64(len(r.buffer)) {
		r.buffer, err = r.next()
		if err != nil {
			return 0, err
		}
		if len(r.buffer) == 0 {
			r.next = r.partStream()
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
func (r *telegramReader) chunk(offset int64, limit int64) ([]byte, error) {

	req := &tg.UploadGetFileRequest{
		Offset:   offset,
		Limit:    int(limit),
		Location: r.location,
	}

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

func (r *telegramReader) partStream() func() ([]byte, error) {

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
		res, err := r.chunk(offset, r.chunkSize)
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
		zap.S().Debugf("Part %d/%d", currentPart, partCount)
		return res, nil
	}
	return readData
}
