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

const (
	oneMB           = 1048576 // 1 MB
	fourKB          = 4096    // 4 KB
	maxFloodWaitSec = 5
)

type chunk struct {
	tag  tg.StorageFileTypeClass
	data []byte
}

type Block struct {
	chunk
	offset   int64
	partSize int
}

func (b Block) Data() []byte {
	return b.data
}

type Reader struct {
	MsgId     int
	sch       schema // immutable
	offset    int64
	offsetMux sync.Mutex
	fileSize  int64
	end       int64
}

func (r *Reader) Next(ctx context.Context, client *tg.Client, loc tg.InputFileLocationClass) (*Block, error) {
	ll := r.getLogger("Next")
	r.offsetMux.Lock()
	if r.offset > r.end {
		ll.Debugf("EOF [r.offset > end (offset=%d, end=%d)]", r.offset, r.end)
		r.offsetMux.Unlock()
		return nil, io.EOF
	}
	offsetSkip := r.offset % fourKB
	offset := r.offset - offsetSkip
	limit, err := r.adjustLimit(offset)
	if err != nil {
		r.offsetMux.Unlock()
		return nil, err
	}
	r.offset = offset + int64(limit)
	r.offsetMux.Unlock()
	ll.Debugf("limit=%d, offset=%d, fileSize=%d, end=%d, offsetSkip=%d", limit, offset, r.fileSize, offset+int64(limit), offsetSkip)
	v, err := r.next(ctx, client, offset, limit, loc)
	if err != nil {
		return nil, err
	}

	offsetEndSkip := int64(len(v.data))
	if offset+int64(len(v.data)) > r.end {
		offsetEndSkip -= (offset + int64(len(v.data)) - r.end - 1)
		ll.Debugf("overshot end (exp=%d vs %d - cut=%d/%d)", r.end, offset+int64(len(v.data)), offsetEndSkip, len(v.data))
	}
	v.data = v.data[offsetSkip:offsetEndSkip]
	return v, nil
}

func (r *Reader) next(ctx context.Context, client *tg.Client, offset int64, limit int, loc tg.InputFileLocationClass) (*Block, error) {
	ll := r.getLogger("next")
	for { // for floodWait and timeout
		if ctx.Err() != nil {
			ll.Debug("context canceled")
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
			return nil, fmt.Errorf("error getting chunk (offset=%d, limit=%d, fileSize=%d, end=%d): %w", offset, limit, r.fileSize, offset+int64(limit), err)
		}

		return &Block{
			chunk:    ch,
			offset:   offset,
			partSize: limit,
		}, nil
	}
}

func (r *Reader) adjustLimit(offset int64) (int, error) {
	ll := r.getLogger("adjustLimit")

	// Valid divisors of 1MB that are multiples of 4KB (powers of 2 from 2^12 to 2^19)
	validLimits := []int{524288, 262144, 131072, 65536, 32768, 16384, 8192, 4096}

	// Calculate maximum possible limit based on fileSize constraint
	maxByFileSize := int(r.fileSize - offset)
	if maxByFileSize <= 0 {
		ll.Debugf("EOF (maxByFileSize <= 0, offset=%d, fileSize=%d)", offset, r.fileSize)
		return 0, io.EOF
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
			return validLimit, nil
		}
	}

	ll.Debugf("no valid limit found, returning least")
	return validLimits[len(validLimits)-1], nil
}
func (r *Reader) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.StreamModule).WithField("func", fmt.Sprintf("%T.%s", r, fn))
}

func NewReader(offset int64, fileSize int64, msgID int, end int64) *Reader {
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
		end:      end,
	}
}
