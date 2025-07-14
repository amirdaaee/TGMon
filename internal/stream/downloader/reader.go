package downloader

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"github.com/sirupsen/logrus"
)

const defaultPartSize = 256 * 1024 // 512 kb
const maxFloodWaitSec = 5

type schema interface {
	Chunk(ctx context.Context, client *tg.Client, offset int64, limit int) (chunk, error)
}

type chunk struct {
	tag  tg.StorageFileTypeClass
	data []byte
}

type Block struct {
	chunk
	offset   int64
	partSize int
}

//	func (b Block) Last() bool {
//		return len(b.data) < b.partSize
//	}
func (b Block) Data() []byte {
	return b.data
}

type Reader struct {
	sch       schema // immutable
	partSize  int    // immutable
	offset    int64
	offsetMux sync.Mutex
	fileSize  int64
}

func (r *Reader) Next(ctx context.Context, client *tg.Client) (*Block, error) {
	r.offsetMux.Lock()
	offset := r.offset
	r.offset += int64(r.partSize)
	r.offsetMux.Unlock()

	return r.next(ctx, client, offset, r.partSize)
}

func (r *Reader) next(ctx context.Context, client *tg.Client, offset int64, limit int) (*Block, error) {
	ll := r.getLogger("next")
	for { // for floodWait and timeout
		if ctx.Err() != nil {
			return nil, nil
		}
		limit = r.adjustLimit(limit, offset)
		ll.Debugf("limit=%d, offset=%d, fileSize=%d", limit, offset, r.fileSize)
		if limit <= 0 {
			return nil, io.EOF
		}
		ch, err := r.sch.Chunk(ctx, client, offset, limit)
		if d, ok := tgerr.AsFloodWait(err); ok {
			sec := d.Seconds()
			ll.WithError(err).Warnf("flood wait %f", sec)
			if sec > maxFloodWaitSec {
				return nil, &ErrFloodWaitTooLong{expected: maxFloodWaitSec, actual: sec}
			}
		}

		if flood, err := tgerr.FloodWait(ctx, err); err != nil {
			if tgerr.Is(err, tg.ErrTimeout) {
				ll.WithError(err).Error("timeout. retrying...")
			}
			if flood || tgerr.Is(err, tg.ErrTimeout) {
				continue
			}
			return nil, fmt.Errorf("error getting chunk: %w", err)
		}

		return &Block{
			chunk:    ch,
			offset:   offset,
			partSize: r.partSize,
		}, nil
	}
}

func (r *Reader) adjustLimit(limit int, offset int64) int {
	ll := r.getLogger("adjustLimit")
	limitDiv := 4 * 1024
	if offset+int64(limit) > r.fileSize {
		limit = int(r.fileSize - offset)
		limit = limit - (limit % limitDiv)
		for {
			if limit <= 0 || 1048576%limit == 0 {
				break
			}
			limit += limitDiv
		}
		ll.Debugf("limit adjusted to %d", limit)
	}
	return limit
}
func (r *Reader) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.StreamModule).WithField("func", fmt.Sprintf("%T.%s", r, fn))
}

func NewReader(offset int64, loc *tg.InputDocumentFileLocation, fileSize int64) *Reader {
	// TODO: client as arg in Next function and passed to master
	master := master{
		precise:  false,
		allowCDN: true,
		location: loc,
	}
	return &Reader{
		sch:      master,
		partSize: defaultPartSize,
		offset:   offset,
		fileSize: fileSize,
	}
}
