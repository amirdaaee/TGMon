package worker

import (
	"fmt"
	"sync"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/tlg"
	"go.uber.org/zap"
)

//go:generate mockgen -source=pool.go -destination=../../mocks/worker/pool.go -package=mocks_worker
type IWorkerPool interface {
	GetNextWorker() IWorker
}

type workerPool struct {
	Bots     []IWorker
	curIndex int
	mut      sync.Mutex
	ll       *zap.Logger
}

var _ IWorkerPool = (*workerPool)(nil)

// GetNextWorker returns the next worker, cycling through available ones.
// It logs which worker index is selected for observability.
func (wp *workerPool) GetNextWorker() IWorker {
	ll := wp.ll.Named("GetNextWorker")
	wp.mut.Lock()
	defer wp.mut.Unlock()

	if len(wp.Bots) == 0 {
		return nil
	}

	wp.curIndex = (wp.curIndex + 1) % len(wp.Bots)
	worker := wp.Bots[wp.curIndex]
	ll.Sugar().Debugf("using worker (%d/%d)", wp.curIndex+1, len(wp.Bots))
	return worker
}

// NewWorkerPool initializes workers concurrently from the provided bot tokens
// and aggregates them into a pool. Returns error if no worker could be started.
func NewWorkerPool(tokens []string, sessCfg *tlg.SessionConfig, channelID int64, cacheRoot string) (IWorkerPool, error) {
	ll := log.GetLogger(log.WorkerModule)
	wp := workerPool{
		ll: ll,
	}
	ll = ll.Named("NewWorkerPool")
	var wg sync.WaitGroup

	for _, token := range tokens {
		wg.Add(1)
		go func(token string) {
			defer wg.Done()
			workerLog := ll.With(zap.String("worker", token[:4]))
			workerLog.Info("initiating worker")

			worker, err := NewWorker(token, sessCfg, channelID, cacheRoot)
			if err != nil {
				workerLog.With(zap.Error(err)).Error("cannot create worker, skipping")
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
