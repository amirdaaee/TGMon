// Package downloader implements chunked file downloads from Telegram, handling
// alignment constraints, chunk sizing, flood waits, and retries.
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

// Block is a unit of downloaded data, capturing the file type tag, raw bytes,
// originating offset, and the requested part size.
type Block struct {
	chunk
	offset   int64
	partSize int
}

// Data returns the payload of the downloaded block.
func (b Block) Data() []byte {
	return b.data
}

// Reader manages sequential retrieval of file chunks for a specific message,
// keeping track of offsets and respecting Telegram download constraints.
type Reader struct {
	MsgId     int
	sch       schema // immutable
	offset    int64
	offsetMux sync.Mutex
	fileSize  int64
	end       int64
}

// Next downloads the next block starting from the internal offset. It aligns
// to 4KB boundaries as required by Telegram, caps the chunk to 1MB boundaries,
// and trims the data to the exact requested range.
func (r *Reader) Next(ctx context.Context, client *tg.Client, loc tg.InputFileLocationClass) (*Block, error) {
	// Check bounds and calculate aligned offset
	r.offsetMux.Lock()
	currentOffset := r.offset
	if currentOffset > r.end {
		r.offsetMux.Unlock()
		r.getLogger("Next").Debugf("EOF: offset %d > end %d", currentOffset, r.end)
		return nil, io.EOF
	}

	// Align offset to 4KB boundary (Telegram requirement)
	offsetSkip := currentOffset % fourKB
	alignedOffset := currentOffset - offsetSkip

	// Calculate optimal limit and update offset
	limit, err := r.adjustLimit(alignedOffset)
	if err != nil {
		r.offsetMux.Unlock()
		return nil, err
	}
	r.offset = alignedOffset + int64(limit)
	r.offsetMux.Unlock()

	// Download chunk
	block, err := r.next(ctx, client, alignedOffset, limit, loc)
	if err != nil {
		return nil, err
	}

	// Trim data to exact requested range
	block.data = r.trimData(block.data, alignedOffset, offsetSkip)
	return block, nil
}

// trimData trims the downloaded data to the exact requested range.
func (r *Reader) trimData(data []byte, alignedOffset int64, offsetSkip int64) []byte {
	endOffset := int64(len(data))

	// Trim end if we overshot
	if alignedOffset+endOffset > r.end {
		trimAmount := alignedOffset + endOffset - r.end - 1
		endOffset -= trimAmount
		r.getLogger("trimData").Debugf("trimming end: %d bytes", trimAmount)
	}

	// Trim start to skip to actual offset
	return data[offsetSkip:endOffset]
}

// next performs the actual chunk request with retry handling for flood waits
// and timeouts. For excessive flood waits, a sentinel error is returned so
// callers can switch workers.
func (r *Reader) next(ctx context.Context, client *tg.Client, offset int64, limit int, loc tg.InputFileLocationClass) (*Block, error) {
	ll := r.getLogger("next")

	for {
		// Check context cancellation
		if ctx.Err() != nil {
			ll.Debug("context canceled")
			return nil, io.EOF
		}

		// Download chunk
		chunk, err := r.sch.Chunk(ctx, client, offset, limit, loc)
		if err != nil {
			// Handle flood wait
			if floodWait, ok := tgerr.AsFloodWait(err); ok {
				waitSeconds := floodWait.Seconds()
				ll.WithError(err).Warnf("flood wait: %.2f seconds", waitSeconds)

				if waitSeconds > maxFloodWaitSec {
					return nil, &ErrFloodWaitTooLong{
						expected: maxFloodWaitSec,
						actual:   waitSeconds,
					}
				}
			}

			// Handle retryable errors (flood wait and timeout)
			if shouldRetry, handledErr := tgerr.FloodWait(ctx, err); handledErr != nil {
				if tgerr.Is(handledErr, tg.ErrTimeout) {
					ll.WithError(handledErr).Warn("timeout, retrying")
				}
				if shouldRetry || tgerr.Is(handledErr, tg.ErrTimeout) {
					continue
				}

				// Non-retryable error
				return nil, fmt.Errorf("error getting chunk (offset=%d, limit=%d, fileSize=%d, end=%d): %w",
					offset, limit, r.fileSize, offset+int64(limit), handledErr)
			}
		}

		// Success
		return &Block{
			chunk:    chunk,
			offset:   offset,
			partSize: limit,
		}, nil
	}
}

// Valid chunk sizes: divisors of 1MB that are multiples of 4KB (powers of 2 from 2^12 to 2^19).
var validChunkSizes = []int{524288, 262144, 131072, 65536, 32768, 16384, 8192, 4096}

// adjustLimit computes an optimal chunk size given the file size and the
// 1MB chunk window boundaries, returning io.EOF if past the end of file.
func (r *Reader) adjustLimit(offset int64) (int, error) {
	// Calculate maximum limit based on remaining file size
	maxByFileSize := int(r.fileSize - offset)
	if maxByFileSize <= 0 {
		r.getLogger("adjustLimit").Debugf("EOF: offset %d >= fileSize %d", offset, r.fileSize)
		return 0, io.EOF
	}

	// Calculate maximum limit to stay within 1MB chunk boundary
	chunkStart := (offset / oneMB) * oneMB
	maxByChunk := int(chunkStart + oneMB - offset)

	// Use the smaller of the two constraints
	maxAllowed := maxByFileSize
	if maxByChunk < maxAllowed {
		maxAllowed = maxByChunk
	}

	// Find the largest valid chunk size that fits
	for _, chunkSize := range validChunkSizes {
		if chunkSize <= maxAllowed {
			r.getLogger("adjustLimit").Debugf("optimal limit: %d (offset=%d, maxAllowed=%d)",
				chunkSize, offset, maxAllowed)
			return chunkSize, nil
		}
	}

	// Fallback to smallest valid size
	return validChunkSizes[len(validChunkSizes)-1], nil
}
func (r *Reader) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.StreamModule).WithField("func", fmt.Sprintf("%T.%s", r, fn))
}

// NewReader constructs a Reader starting at offset up to end, using the master
// schema by default and enabling CDN when allowed by Telegram.
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
