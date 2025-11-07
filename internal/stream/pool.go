// Package stream provides a pool of Telegram workers that cooperatively
// stream files with backoff on flood waits and timeouts.
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

// IWorkerPool exposes worker round-robin selection and stream creation.
//
//go:generate mockgen -source=pool.go -destination=../../mocks/stream/pool.go -package=mocks
type IWorkerPool interface {
	// GetNextWorker returns the next worker in a round-robin fashion.
	GetNextWorker() IWorker
	// Stream creates a buffered reader that streams document content from
	// Telegram using the pool for resiliency.
	Stream(ctx context.Context, msgID int, offset int64, end int64) (IStreamer, error)
}
type workerPool struct {
	Bots     []IWorker
	curIndex int
	mut      sync.Mutex
}

var _ IWorkerPool = (*workerPool)(nil)

// GetNextWorker returns the next worker, cycling through available ones.
// It logs which worker index is selected for observability.
func (wp *workerPool) GetNextWorker() IWorker {
	wp.mut.Lock()
	defer wp.mut.Unlock()

	if len(wp.Bots) == 0 {
		return nil
	}

	wp.curIndex = (wp.curIndex + 1) % len(wp.Bots)
	worker := wp.Bots[wp.curIndex]
	wp.getLogger("GetNextWorker").Debugf("using worker (%d/%d)", wp.curIndex+1, len(wp.Bots))
	return worker
}

// Stream constructs a new Streamer over the pool for the specified message
// and byte range [offset, end].
func (wp *workerPool) Stream(ctx context.Context, msgID int, offset int64, end int64) (IStreamer, error) {
	return NewStreamer(ctx, wp, msgID, offset, end)

}
func (wp *workerPool) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.StreamModule).WithField("func", fmt.Sprintf("%T.%s", wp, fn))
}

// NewWorkerPool initializes workers concurrently from the provided bot tokens
// and aggregates them into a pool. Returns error if no worker could be started.
func NewWorkerPool(tokens []string, sessCfg *tlg.SessionConfig, channelID int64, cacheRoot string) (IWorkerPool, error) {
	ll := log.GetLogger(log.StreamModule).WithField("func", "NewWorkerPool")
	wp := workerPool{}
	var wg sync.WaitGroup

	for _, token := range tokens {
		wg.Add(1)
		go func(token string) {
			defer wg.Done()
			workerLog := ll.WithField("worker", token)
			workerLog.Info("initiating worker")

			worker, err := NewWorker(token, sessCfg, channelID, cacheRoot)
			if err != nil {
				workerLog.WithError(err).Error("cannot create worker, skipping")
				return
			}

			wp.mut.Lock()
			wp.Bots = append(wp.Bots, worker)
			wp.mut.Unlock()

			workerLog.Info("worker initiated")
		}(token)
	}

	wg.Wait()
	if len(wp.Bots) == 0 {
		return nil, fmt.Errorf("no workers available")
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

// Read implements io.Reader, reading from the current worker. If a flood wait
// is encountered, it switches to the next worker transparently. Leftover bytes
// from larger chunks are preserved and returned first on the next Read.
func (s *Streamer) Read(p []byte) (n int, err error) {
	// Return leftover bytes if available
	if len(s.leftover) > 0 {
		n := copy(p, s.leftover)
		s.leftover = s.leftover[n:]
		return n, nil
	}

	// Try to get data from workers
	for {
		worker := s.wp.GetNextWorker()
		data, err := worker.Stream(s.ctx, s.reader)

		if err != nil {
			if errors.Is(err, &downloader.ErrFloodWaitTooLong{}) {
				s.getLogger("Read").Warn("flood wait too long, trying next worker")
				continue
			}
			if errors.Is(err, io.EOF) {
				s.getLogger("Read").Debug("end of file reached")
				return 0, io.EOF
			}
			return 0, fmt.Errorf("error streaming: %w", err)
		}

		// Copy data to buffer, save leftover if needed
		n := copy(p, data)
		if n < len(data) {
			s.leftover = append(s.leftover[:0], data[n:]...)
		}
		return n, nil
	}
}

// GetBuffer returns the sized bufio.Reader created for this Streamer.
func (s *Streamer) GetBuffer() *bufio.Reader {
	return s.buff
}
func (s *Streamer) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.StreamModule).WithField("func", fmt.Sprintf("%T.%s", s, fn))
}

// NewStreamer prepares a downloader.Reader for the target document and wraps
// it into a Streamer buffered according to runtime configuration.
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
