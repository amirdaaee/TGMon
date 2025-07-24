package stream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/stream/downloader"
	"github.com/amirdaaee/TGMon/internal/tlg"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

//go:generate mockgen -source=pool.go -destination=../../mocks/stream/pool.go -package=mocks
type IWorkerPool interface {
	GetNextWorker() IWorker
	Stream(ctx context.Context, msgID int, offset int64, writer io.Writer) error
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
func (wp *workerPool) Stream(ctx context.Context, msgID int, offset int64, writer io.Writer) error {
	ll := wp.getLogger("Stream")
	doc, err := wp.GetNextWorker().GetDoc(ctx, msgID)
	if err != nil {
		return fmt.Errorf("error getting doc: %w", err)
	}
	dataChan := make(chan *downloader.Block, 10)
	reader := downloader.NewReader(offset, doc.GetSize(), msgID)
	errG, ctx := errgroup.WithContext(ctx)
	errG.Go(func() error {
		ll.Debug("starting stream")
		if err := wp.startStream(ctx, dataChan, reader); err != nil {
			return fmt.Errorf("error starting stream: %w", err)
		}
		return nil
	})
	errG.Go(func() error {
		ll.Debug("starting write buffer")
		if err := wp.writeBuffer(ctx, writer, dataChan); err != nil {
			return fmt.Errorf("error writing buffer: %w", err)
		}
		return nil
	})
	if err := errG.Wait(); err != nil {
		if errors.Is(err, io.EOF) {
			ll.Debug("end of file reached")
			return nil
		} else {
			return fmt.Errorf("error streaming: %w", err)
		}
	}
	ll.Debug("stream finished")
	return nil
}
func (wp *workerPool) writeBuffer(ctx context.Context, writer io.Writer, dataChan <-chan *downloader.Block) error {
	ll := wp.getLogger("writeBuffer")
	for {
		select {
		case <-ctx.Done():
			return nil
		case block := <-dataChan:
			if block == nil {
				ll.Debug("nil block received. stopping write buffer")
				return nil
			}
			if _, err := writer.Write(block.Data()); err != nil {
				return fmt.Errorf("error writing to buffer: %w", err)
			}
		}
	}
}
func (wp *workerPool) startStream(ctx context.Context, dataChan chan *downloader.Block, reader *downloader.Reader) error {
	ll := wp.getLogger("startStream")
	defer close(dataChan)
	for {
		wrkr := wp.GetNextWorker()
		if err := wrkr.Stream(ctx, reader, dataChan); err != nil {
			if errors.Is(err, &downloader.ErrFloodWaitTooLong{}) {
				ll.Warn("flood wait. using next worker")
				continue
			} else {
				return fmt.Errorf("error streaming: %w", err)
			}
		}
	}
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
