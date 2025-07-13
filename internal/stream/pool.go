package stream

import (
	"fmt"
	"sync"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/tlg"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -source=pool.go -destination=../../mocks/stream/pool.go -package=mocks
type IWorkerPool interface {
	GetNextWorker() IWorker
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
