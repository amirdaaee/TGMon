package downloader

import (
	"context"
	"fmt"
	"strconv"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/go-faster/errors"
	"github.com/sirupsen/logrus"

	"github.com/gotd/td/tg"
)

// RedirectError error is returned when Downloader get CDN redirect.
// See https://core.telegram.org/constructor/upload.fileCdnRedirect.
type RedirectError struct {
	Redirect *tg.UploadFileCDNRedirect
}

// Error implements error interface.
func (r *RedirectError) Error() string {
	return "redirect to CDN DC " + strconv.Itoa(r.Redirect.DCID)
}

// master is a master DC download schema.
// See https://core.telegram.org/api/files#downloading-files.
type master struct {
	precise  bool
	allowCDN bool
	location *tg.InputDocumentFileLocation
}

var _ schema = master{}

func (c master) Chunk(ctx context.Context, client *tg.Client, offset int64, limit int) (chunk, error) {
	ll := c.getLogger("Chunk")
	ll.Debugf("getting chunk from offset %d with limit %d", offset, limit)
	req := &tg.UploadGetFileRequest{
		Offset:   offset,
		Limit:    limit,
		Location: c.location,
	}
	req.SetCDNSupported(c.allowCDN)
	req.SetPrecise(c.precise)
	r, err := client.UploadGetFile(ctx, req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			ll.Debug("context canceled. returning empty chunk")
			return chunk{}, nil
		}
		return chunk{}, err
	}

	switch result := r.(type) {
	case *tg.UploadFile:
		return chunk{data: result.Bytes, tag: result.Type}, nil
	case *tg.UploadFileCDNRedirect:
		return chunk{}, &RedirectError{Redirect: result}
	default:
		return chunk{}, errors.Errorf("unexpected type %T", r)
	}
}

func (c *master) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.StreamModule).WithField("func", fmt.Sprintf("%T.%s", c, fn))
}
