package filesystem

import (
	"context"
	"fmt"
	"io"
	"syscall"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/stream"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/sirupsen/logrus"
)

// MediaFileHandle handles read operations on a media file
type MediaFileHandle struct {
	media            *types.MediaFileDoc
	streamWorkerPool stream.IWorkerPool
	ctx              context.Context
	cancel           context.CancelFunc
}

var _ fs.FileReader = (*MediaFileHandle)(nil)
var _ fs.FileReleaser = (*MediaFileHandle)(nil)

// Read reads data from the file at the specified offset
func (mfh *MediaFileHandle) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	ll := mfh.getLogger("Read")
	ll.Debugf("Read request: offset=%d, size=%d, fileSize=%d", off, len(dest), mfh.media.Meta.FileSize)

	// Check if offset is beyond file size
	if off >= mfh.media.Meta.FileSize {
		ll.Debug("EOF: offset beyond file size")
		return fuse.ReadResultData(nil), 0
	}

	// Calculate how much to read
	toRead := int64(len(dest))
	if off+toRead > mfh.media.Meta.FileSize {
		toRead = mfh.media.Meta.FileSize - off
	}

	// Create a new streamer for this read operation with the correct offset
	// This allows seeking to any position in the file
	end := off + toRead - 1
	if end >= mfh.media.Meta.FileSize {
		end = mfh.media.Meta.FileSize - 1
	}

	// Create a context that is canceled when either:
	// 1. The request context (ctx) is canceled (user interrupt, FUSE connection close)
	// 2. The file handle context (mfh.ctx) is canceled (file is closed)
	// This ensures stream operations are canceled in both cases.
	// The goroutine exits when any of the contexts are canceled, preventing leaks.
	streamCtx, streamCancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-mfh.ctx.Done():
			// File handle was closed, cancel the stream operation
			streamCancel()
		case <-ctx.Done():
			// Request was canceled, streamCancel will be called by defer
			// This case ensures the goroutine doesn't block if ctx is canceled
		case <-streamCtx.Done():
			// Stream context already canceled (e.g., Read returned normally)
		}
	}()
	defer streamCancel() // Ensure stream context is canceled when Read returns

	// Use the combined context for stream operations
	streamer, err := mfh.streamWorkerPool.Stream(streamCtx, mfh.media.MessageID, off, end)
	if err != nil {
		ll.WithError(err).Error("Failed to create streamer")
		return nil, syscall.EIO
	}

	// Read the data
	data := make([]byte, toRead)
	totalRead := int64(0)
	for totalRead < toRead {
		n, err := streamer.Read(data[totalRead:])
		if err != nil && err != io.EOF {
			ll.WithError(err).Error("Failed to read from streamer")
			return nil, syscall.EIO
		}
		if n == 0 {
			break
		}
		totalRead += int64(n)
		if err == io.EOF {
			break
		}
	}

	// Trim to actual read size
	if totalRead < toRead {
		data = data[:totalRead]
	}
	ll.Debugf("Read %d bytes", totalRead)

	return fuse.ReadResultData(data), 0
}

// Release is called when the file is closed/released
func (mfh *MediaFileHandle) Release(ctx context.Context) syscall.Errno {
	ll := mfh.getLogger("Release")
	ll.Debug("File handle released, canceling context")
	if mfh.cancel != nil {
		mfh.cancel()
	}
	return 0
}

func (mfh *MediaFileHandle) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.FuseModule).WithField("func", fmt.Sprintf("%T.%s", mfh, fn))
}
