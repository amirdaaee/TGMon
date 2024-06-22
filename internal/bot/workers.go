package bot

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amirdaaee/TGMon/config"
	"github.com/celestix/gotgproto"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"
)

var Workers *BotWorkers = &BotWorkers{
	log:  nil,
	Bots: make([]*Worker, 0),
}

func GetNextWorker() *Worker {
	Workers.mut.Lock()
	defer Workers.mut.Unlock()
	index := (Workers.index + 1) % len(Workers.Bots)
	Workers.index = index
	worker := Workers.Bots[index]
	Workers.log.Sugar().Debugf("Using worker %d", worker.ID)
	return worker
}

func StartWorkers(log *zap.Logger) (*BotWorkers, error) {
	Workers.Init(log)

	if len(config.Config().WorkerTokens) == 0 {
		Workers.log.Sugar().Info("No worker bot tokens provided, skipping worker initialization")
		return Workers, nil
	}
	Workers.log.Sugar().Info("Starting")
	newpath := filepath.Join(".", config.Config().SessionDir)
	if err := os.MkdirAll(newpath, os.ModePerm); err != nil {
		Workers.log.Error("Failed to create sessions directory", zap.Error(err))
		return nil, err
	}

	var wg sync.WaitGroup
	var successfulStarts int32
	totalBots := len(config.Config().WorkerTokens)

	for i := 0; i < totalBots; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			done := make(chan error, 1)
			go func() {
				err := Workers.Add(config.Config().WorkerTokens[i])
				done <- err
			}()

			select {
			case err := <-done:
				if err != nil {
					Workers.log.Error("Failed to start worker", zap.Int("index", i), zap.Error(err))
				} else {
					atomic.AddInt32(&successfulStarts, 1)
				}
			case <-ctx.Done():
				Workers.log.Error("Timed out starting worker", zap.Int("index", i))
			}
		}(i)
	}

	wg.Wait() // Wait for all goroutines to finish
	Workers.log.Sugar().Infof("Successfully started %d/%d bots", successfulStarts, totalBots)
	return Workers, nil
}

func NewTelegramReader(
	ctx context.Context,
	client *gotgproto.Client,
	location *tg.InputDocumentFileLocation,
	start int64,
	end int64,
	contentLength int64,
) (io.ReadCloser, error) {

	r := &telegramReader{
		ctx:           ctx,
		location:      location,
		client:        client,
		start:         start,
		end:           end,
		chunkSize:     int64(1024 * 1024),
		contentLength: contentLength,
	}
	zap.S().Debug("Start")
	r.next = r.partStream()
	return r, nil
}

func DeleteMessage(w *Worker, msgID int) error {
	return w.Client.CreateContext().DeleteMessages(config.Config().ChannelID, []int{msgID})
}

func GetThumbnail(ctx context.Context, loc *tg.InputDocumentFileLocation, size string, sizrB int) ([]byte, error) {
	loc_ := tg.InputDocumentFileLocation{}
	loc_.FillFrom(loc)
	loc_.ThumbSize = size
	req := &tg.UploadGetFileRequest{
		Location: &loc_,
		Limit:    1024 * 1024,
		Precise:  false,
	}
	worker := GetNextWorker()
	res, err := worker.Client.API().UploadGetFile(ctx, req)
	if err != nil {
		return nil, err
	}
	thumbFile := res.(*tg.UploadFile)
	return thumbFile.Bytes, nil
}
