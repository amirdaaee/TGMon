package stream

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/amirdaaee/TGMon/internal/config"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/stream/downloader"
	"github.com/amirdaaee/TGMon/internal/tlg"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -source=pool.go -destination=../../mocks/stream/pool.go -package=mocks
type IWorkerPool interface {
	GetNextWorker() IWorker
	Stream(ctx context.Context, msgID int, offset int64, end int64) (IStreamer, error)
}
type workerPool struct {
	Bots     []IWorker
	curIndex int
	mut      sync.Mutex
}

var _ IWorkerPool = (*workerPool)(nil)

func (wp *workerPool) GetNextWorker() IWorker {
	if len(wp.Bots) == 0 {
		return nil
	}
	wp.mut.Lock()
	defer wp.mut.Unlock()
	index := (wp.curIndex + 1) % len(wp.Bots)
	wp.curIndex = index
	worker := wp.Bots[index]
	wp.getLogger("GetNextWorker").Debugf("using worker (%d/%d)", index+1, len(wp.Bots))
	return worker
}
func (wp *workerPool) Stream(ctx context.Context, msgID int, offset int64, end int64) (IStreamer, error) {
	return NewStreamer(ctx, wp, msgID, offset, end)

}
func (wp *workerPool) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.StreamModule).WithField("func", fmt.Sprintf("%T.%s", wp, fn))
}

func NewWorkerPool(tokens []string, sessCfg *tlg.SessionConfig, channelID int64, cacheRoot string) (IWorkerPool, error) {
	ll := log.GetLogger(log.StreamModule).WithField("func", "NewWorkerPool")
	wp := workerPool{}
	var wg sync.WaitGroup
	for _, tok := range tokens {
		wg.Add(1)
		go func(_i string) {
			defer wg.Done()
			ll := ll.WithField("worker", _i)
			ll.Info("initiating worker")
			w, err := NewWorker(tok, sessCfg, channelID, cacheRoot)
			if err != nil {
				ll.WithError(err).Error("can not create worker. skipping ...")
				return
			}
			wp.mut.Lock()
			defer wp.mut.Unlock()
			wp.Bots = append(wp.Bots, w)
			ll.Info("worker initaited")
		}(tok)
	}
	wg.Wait()
	if len(wp.Bots) == 0 {
		return nil, fmt.Errorf("no worker is avaiable")
	}
	return &wp, nil
}

type IStreamer interface {
	io.Reader
	GetBuffer() *bufio.Reader
}
type Streamer struct {
	ctx      context.Context
	msgID    int
	offset   int64
	reader   *downloader.Reader
	wp       IWorkerPool
	buff     *bufio.Reader
	leftover []byte
}

var _ IStreamer = (*Streamer)(nil)

func (s *Streamer) Read(p []byte) (n int, err error) {
	ll := s.getLogger("startStream")
	if len(s.leftover) > 0 {
		n := copy(p, s.leftover)
		s.leftover = s.leftover[n:]
		return n, nil
	}
	for {
		wrkr := s.wp.GetNextWorker()
		v, err := wrkr.Stream(s.ctx, s.reader)
		if err != nil {
			if errors.Is(err, &downloader.ErrFloodWaitTooLong{}) {
				ll.Warn("flood wait. using next worker")
				continue
			} else if errors.Is(err, io.EOF) {
				ll.Debug("end of file reached (io.EOF)")
				return 0, io.EOF
			} else {
				return 0, fmt.Errorf("error streaming: %w", err)
			}
		}
		n := copy(p, v)
		if n < len(v) {
			s.leftover = append(s.leftover[:0], v[n:]...)
		}
		return n, nil
	}
}
func (s *Streamer) GetBuffer() *bufio.Reader {
	return s.buff
}
func (s *Streamer) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.StreamModule).WithField("func", fmt.Sprintf("%T.%s", s, fn))
}
func NewStreamer(ctx context.Context, wp IWorkerPool, msgID int, offset int64, end int64) (*Streamer, error) {
	doc, err := wp.GetNextWorker().GetDoc(ctx, msgID)
	if err != nil {
		return nil, fmt.Errorf("error getting doc: %w", err)
	}
	reader := downloader.NewReader(offset, doc.GetSize(), msgID, end)
	v := &Streamer{
		ctx:    ctx,
		msgID:  msgID,
		offset: offset,
		reader: reader,
		wp:     wp,
	}
	v.buff = bufio.NewReaderSize(v, config.Config().RuntimeConfig.StreamBuffSize)
	return v, nil
}
