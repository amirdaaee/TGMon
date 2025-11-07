// Package stream provides Telegram file streaming utilities built around a pool
// of workers. Each worker talks to Telegram APIs to fetch documents, thumbnails
// and file chunks, applying caching and resiliency to rate limits.
package stream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/amirdaaee/TGMon/internal/stream/downloader"
	"github.com/amirdaaee/TGMon/internal/tlg"
	"github.com/celestix/gotgproto"
	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
)

// IWorker encapsulates Telegram document operations for a single bot/account.
// Implementations fetch document metadata, thumbnails, and stream file blocks.
//
//go:generate mockgen -source=worker.go -destination=../../mocks/stream/worker.go -package=mocks
type IWorker interface {
	// GetThumbnail returns the first available thumbnail bytes for a document
	// in the specified message.
	GetThumbnail(ctx context.Context, messageID int) ([]byte, error)
	// GetDoc returns the Telegram document of a message, possibly using cache.
	GetDoc(ctx context.Context, messageID int) (*tg.Document, error)
	// Stream fetches the next block using the provided downloader.Reader.
	Stream(ctx context.Context, reader *downloader.Reader) ([]byte, error)
}
type worker struct {
	cl            tlg.IClient
	channelID     int64
	cache         IFileCache[int64]
	docCache      IFileCache[[]byte]
	tgChannel     tg.InputChannelClass
	tgChannelLock sync.Mutex
}

var _ IWorker = (*worker)(nil)

// thumbnailLimit caps thumbnail downloads to 1MB which is sufficient for
// Telegram thumbnails and keeps network usage bounded.
const thumbnailLimit = 1024 * 1024 // 1 MB

// GetThumbnail downloads the first available thumbnail for the document inside
// the given channel message. It ensures the access hash is up-to-date before
// requesting the thumbnail file from Telegram.
func (w *worker) GetThumbnail(ctx context.Context, messageID int) ([]byte, error) {
	doc, err := w.GetDoc(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("error getting document: %w", err)
	}

	// Get thumbnail size
	thumbs, ok := doc.GetThumbs()
	if !ok || len(thumbs) == 0 {
		return nil, ErrNoThumbnail
	}

	thumbSize, ok := thumbs[0].(*tg.PhotoSize)
	if !ok {
		return nil, &tlg.UnexpectedTypeErrType{ExpectedType: &tg.PhotoSize{}, GotType: thumbs[0]}
	}

	// Ensure access hash is cached
	if _, err := w.getDocAccHash(ctx, messageID); err != nil {
		return nil, fmt.Errorf("error updating access hash: %w", err)
	}

	// Create file location request
	location := tg.InputDocumentFileLocation{}
	location.FillFrom(doc.AsInputDocumentFileLocation())
	location.ThumbSize = thumbSize.Type

	req := &tg.UploadGetFileRequest{
		Location: &location,
		Limit:    thumbnailLimit,
		Precise:  false,
	}

	// Download thumbnail
	result, err := w.getTgApi().UploadGetFile(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error downloading thumbnail: %w", err)
	}

	thumbFile, ok := result.(*tg.UploadFile)
	if !ok {
		return nil, &tlg.UnexpectedTypeErrType{ExpectedType: &tg.UploadFile{}, GotType: result}
	}

	return thumbFile.GetBytes(), nil
}

// GetDoc fetches the message document and caches its encoded bytes on disk.
// Subsequent calls return the cached value to reduce API calls.
func (w *worker) GetDoc(ctx context.Context, messageID int) (*tg.Document, error) {
	cacheName := w.cacheNamePrefix(messageID)
	dataRaw, err := w.docCache.GetOrSet(cacheName, func() ([]byte, error) {
		doc, err := w.retrieveChannelMessageDoc(ctx, messageID)
		if err != nil {
			return nil, fmt.Errorf("error getting document of message: %w", err)
		}
		resBuf := bin.Buffer{Buf: []byte{}}
		if err := doc.Encode(&resBuf); err != nil {
			return nil, fmt.Errorf("error encoding document: %w", err)
		}
		return resBuf.Buf, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error getting document from cache: %w", err)
	}
	doc := tg.Document{}
	if err := doc.Decode(&bin.Buffer{Buf: dataRaw}); err != nil {
		return nil, fmt.Errorf("error decoding document: %w", err)
	}
	return &doc, nil
}

// Stream retrieves the next data block via the provided downloader.Reader.
// It resolves the document location once and then pulls the next chunk.
func (w *worker) Stream(ctx context.Context, reader *downloader.Reader) ([]byte, error) {
	doc, err := w.GetDoc(ctx, reader.MsgId)
	if err != nil {
		return nil, fmt.Errorf("error getting document: %w", err)
	}

	location := doc.AsInputDocumentFileLocation()
	block, err := reader.Next(ctx, w.getTgApi(), location)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.EOF
		}
		return nil, fmt.Errorf("error getting block: %w", err)
	}

	if block == nil {
		return nil, io.EOF
	}

	return block.Data(), nil
}
func (w *worker) getChannel(ctx context.Context) (tg.InputChannelClass, error) {
	w.tgChannelLock.Lock()
	defer w.tgChannelLock.Unlock()
	if w.tgChannel == nil {
		channel, err := w.retrieveChannel(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting channel: %w", err)
		}
		w.tgChannel = channel
	}
	return w.tgChannel, nil
}
func (w *worker) getDocAccHash(ctx context.Context, messageID int) (int64, error) {
	cacheName := w.cacheNamePrefix(messageID)
	return w.cache.GetOrSet(cacheName, func() (int64, error) {
		return w.retrieveAccHash(ctx, messageID)
	})
}

func (w *worker) retrieveAccHash(ctx context.Context, messageID int) (int64, error) {
	doc, err := w.retrieveChannelMessageDoc(ctx, messageID)
	if err != nil {
		return 0, fmt.Errorf("error getting document of message: %w", err)
	}
	return doc.GetAccessHash(), nil
}
func (w *worker) getTg() *gotgproto.Client {
	return w.cl.GetClient()
}
func (w *worker) getTgApi() *tg.Client {
	return w.cl.GetClient().API()
}
func (w *worker) retrieveChannel(ctx context.Context) (tg.InputChannelClass, error) {
	api := w.getTgApi()
	inputChannel := &tg.InputChannel{ChannelID: w.channelID}
	chatList, err := api.ChannelsGetChannels(ctx, []tg.InputChannelClass{inputChannel})
	if err != nil {
		return nil, fmt.Errorf("cannot list channels: %w", err)
	}

	chats := chatList.GetChats()
	switch len(chats) {
	case 0:
		return nil, fmt.Errorf("channel not found")
	case 1:
		// Expected case
	default:
		return nil, fmt.Errorf("multiple channels found (expected 1, got %d)", len(chats))
	}

	channel, ok := chats[0].(*tg.Channel)
	if !ok {
		return nil, &tlg.UnexpectedTypeErrType{ExpectedType: &tg.Channel{}, GotType: chats[0]}
	}

	return channel.AsInput(), nil
}
func (w *worker) retrieveChannelMessage(ctx context.Context, messageID int) (tg.MessageClass, error) {
	channel, err := w.getChannel(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting channel: %w", err)
	}

	inputMsg := []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}}
	response, err := w.getTgApi().ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
		Channel: channel,
		ID:      inputMsg,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting message: %w", err)
	}

	channelMessages, ok := response.(*tg.MessagesChannelMessages)
	if !ok {
		return nil, &tlg.UnexpectedTypeErrType{ExpectedType: &tg.MessagesChannelMessages{}, GotType: response}
	}

	messages := channelMessages.Messages
	switch len(messages) {
	case 0:
		return nil, fmt.Errorf("message not found")
	case 1:
		return messages[0], nil
	default:
		return nil, fmt.Errorf("multiple messages found (expected 1, got %d)", len(messages))
	}
}
func (w *worker) retrieveChannelMessageDoc(ctx context.Context, messageID int) (*tg.Document, error) {
	message, err := w.retrieveChannelMessage(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("error getting message: %w", err)
	}

	msg, ok := message.(*tg.Message)
	if !ok {
		return nil, &tlg.UnexpectedTypeErrType{ExpectedType: &tg.Message{}, GotType: message}
	}

	media, ok := msg.Media.(*tg.MessageMediaDocument)
	if !ok {
		return nil, &tlg.UnexpectedTypeErrType{ExpectedType: &tg.MessageMediaDocument{}, GotType: msg.Media}
	}

	doc, ok := media.Document.(*tg.Document)
	if !ok {
		return nil, &tlg.UnexpectedTypeErrType{ExpectedType: &tg.Document{}, GotType: media.Document}
	}

	return doc, nil
}
func (w *worker) cacheNamePrefix(s int) string {
	return fmt.Sprintf("%d-%d", w.getTg().Self.GetID(), s)
}

//	func (w *worker) getLogger(fn string) *logrus.Entry {
//		return log.GetLogger(log.StreamModule).WithField("func", fmt.Sprintf("%T.%s", w, fn))
//	}
func NewWorker(token string, sessCfg *tlg.SessionConfig, channelID int64, cacheRoot string) (IWorker, error) {
	w := worker{
		cl:        tlg.NewTgClient(sessCfg, token),
		channelID: channelID,
		cache:     NewAccessHashCache(cacheRoot),
		docCache:  NewDocCache(cacheRoot),
	}

	if err := w.cl.Connect(); err != nil {
		return nil, fmt.Errorf("cannot connect to worker: %w", err)
	}

	return &w, nil
}
