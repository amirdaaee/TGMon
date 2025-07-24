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

const maxFloodWaitSec = 5

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
	MsgId     int
	sch       schema // immutable
	offset    int64
	offsetMux sync.Mutex
	fileSize  int64
}

func (r *Reader) Next(ctx context.Context, client *tg.Client, loc tg.InputFileLocationClass) (*Block, error) {
	ll := r.getLogger("Next")
	r.offsetMux.Lock()
	limit := r.adjustLimit(r.offset)
	offset := r.offset
	r.offset += int64(limit)
	r.offsetMux.Unlock()
	ll.Debugf("limit=%d, offset=%d, fileSize=%d", limit, offset, r.fileSize)
	return r.next(ctx, client, offset, limit, loc)
}

func (r *Reader) next(ctx context.Context, client *tg.Client, offset int64, limit int, loc tg.InputFileLocationClass) (*Block, error) {
	ll := r.getLogger("next")
	for { // for floodWait and timeout
		if ctx.Err() != nil {
			return nil, nil
		}
		if limit <= 0 {
			return nil, io.EOF
		}
		ch, err := r.sch.Chunk(ctx, client, offset, limit, loc)
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
			partSize: limit,
		}, nil
	}
}

func (r *Reader) adjustLimit(offset int64) int {
	ll := r.getLogger("adjustLimit")
	const (
		oneMB  = 1048576 // 1 MB
		fourKB = 4096    // 4 KB
	)

	// Valid divisors of 1MB that are multiples of 4KB (powers of 2 from 2^12 to 2^19)
	validLimits := []int{524288, 262144, 131072, 65536, 32768, 16384, 8192, 4096}

	// Calculate maximum possible limit based on fileSize constraint
	maxByFileSize := int(r.fileSize - offset)
	if maxByFileSize <= 0 {
		ll.Warnf("maxByFileSize <= 0, offset=%d, fileSize=%d", offset, r.fileSize)
		return 0
	}

	// Calculate maximum possible limit to stay within 1MB chunk from beginning
	chunkStart := (offset / oneMB) * oneMB
	maxByChunk := int(chunkStart + oneMB - offset)

	// Find the largest valid limit that satisfies all constraints
	maxAllowed := maxByFileSize
	if maxByChunk < maxAllowed {
		maxAllowed = maxByChunk
	}

	// Find the largest valid limit from our predefined list
	for _, validLimit := range validLimits {
		if validLimit <= maxAllowed {
			ll.Debugf("optimal limit found: %d (offset=%d, fileSize=%d, maxByFileSize=%d, maxByChunk=%d)",
				validLimit, offset, r.fileSize, maxByFileSize, maxByChunk)
			return validLimit
		}
	}

	ll.Warnf("no valid limit found, returning least")
	return validLimits[len(validLimits)-1]
}
func (r *Reader) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.StreamModule).WithField("func", fmt.Sprintf("%T.%s", r, fn))
}

func NewReader(offset int64, fileSize int64, msgID int) *Reader {
	// TODO: client as arg in Next function and passed to master
	master := master{
		precise:  false,
		allowCDN: true,
	}
	return &Reader{
		sch:      master,
		offset:   offset,
		fileSize: fileSize,
		MsgId:    msgID,
	}
}
