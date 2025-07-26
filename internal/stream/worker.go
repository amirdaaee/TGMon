package stream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/stream/downloader"
	"github.com/amirdaaee/TGMon/internal/tlg"
	"github.com/celestix/gotgproto"
	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -source=worker.go -destination=../../mocks/stream/worker.go -package=mocks
type IWorker interface {
	GetThumbnail(ctx context.Context, messageID int) ([]byte, error)
	GetDoc(ctx context.Context, messageID int) (*tg.Document, error)
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

func (w *worker) GetThumbnail(ctx context.Context, messageID int) ([]byte, error) {
	doc, err := w.GetDoc(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("error getting document: %w", err)
	}
	thumbs, ok := doc.GetThumbs()
	if !ok {
		return nil, ErrNoThumbnail
	}
	thmb, ok := thumbs[0].(*tg.PhotoSize)
	if !ok {
		return nil, &tlg.UnexpectedTypeErrType{ExpectedType: &tg.PhotoSize{}, GotType: thumbs[0]}
	}
	size := thmb.Type
	loc_ := tg.InputDocumentFileLocation{}

	if _, err := w.getDocAccHash(ctx, messageID); err != nil {
		return nil, fmt.Errorf("error updating access hash: %s", err)
	}
	loc_.FillFrom(doc.AsInputDocumentFileLocation())
	loc_.ThumbSize = size
	req := &tg.UploadGetFileRequest{
		Location: &loc_,
		Limit:    1024 * 1024,
		Precise:  false,
	}
	res, err := w.getTgApi().UploadGetFile(ctx, req)
	if err != nil {
		return nil, err
	}
	thumbFile, ok := res.(*tg.UploadFile)
	if !ok {
		return nil, &tlg.UnexpectedTypeErrType{ExpectedType: &tg.UploadFile{}, GotType: res}
	}
	return thumbFile.GetBytes(), nil
}
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
func (w *worker) Stream(ctx context.Context, reader *downloader.Reader) ([]byte, error) {
	ll := w.getLogger("Stream")
	doc, err := w.GetDoc(ctx, reader.MsgId)
	if err != nil {
		return nil, fmt.Errorf("error getting document: %w", err)
	}
	inDoc := doc.AsInputDocumentFileLocation()
	block, err := reader.Next(ctx, w.getTgApi(), inDoc)
	if err != nil {
		if errors.Is(err, io.EOF) {
			ll.Debug("end of file reached (io.EOF)")
			return nil, io.EOF
		}
		return nil, fmt.Errorf("error getting block: %w", err)
	}
	if block == nil {
		ll.Debug("end of file reached (nil block)")
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
	chatList, err := api.ChannelsGetChannels(ctx, []tg.InputChannelClass{&tg.InputChannel{ChannelID: w.channelID}})
	if err != nil {
		return nil, fmt.Errorf("can not list channels: %w", err)
	}
	if len(chatList.GetChats()) == 0 {
		return nil, fmt.Errorf("channel not found")
	} else if len(chatList.GetChats()) > 1 {
		return nil, fmt.Errorf("multiple channels found")
	}
	cht := chatList.GetChats()[0]
	var channel tg.InputChannelClass
	if chn, ok := cht.(*tg.Channel); !ok {
		return nil, &tlg.UnexpectedTypeErrType{ExpectedType: &tg.Channel{}, GotType: chn}
	} else {
		channel = chn.AsInput()
	}
	return channel, nil
}
func (w *worker) retrieveChannelMessage(ctx context.Context, messageID int) (tg.MessageClass, error) {
	channel, err := w.getChannel(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting channel: %w", err)
	}
	inputMsgList := []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}}
	chennelMsgCls, err := w.getTgApi().ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
		Channel: channel,
		ID:      inputMsgList,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting message of document: %s", err)
	}
	chennelMsg, ok := chennelMsgCls.(*tg.MessagesChannelMessages)
	if !ok {
		return nil, &tlg.UnexpectedTypeErrType{ExpectedType: &tg.MessagesChannelMessages{}, GotType: chennelMsg}
	}
	if len(chennelMsg.Messages) == 0 {
		return nil, fmt.Errorf("message not found")
	} else if len(chennelMsg.Messages) > 1 {
		return nil, fmt.Errorf("multiple messages found")
	}
	msgCls := chennelMsg.Messages[0]
	return msgCls, nil
}
func (w *worker) retrieveChannelMessageDoc(ctx context.Context, messageID int) (*tg.Document, error) {
	msgCls, err := w.retrieveChannelMessage(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("error getting message of document: %w", err)
	}
	msg, ok := msgCls.(*tg.Message)
	if !ok {
		return nil, &tlg.UnexpectedTypeErrType{ExpectedType: &tg.Message{}, GotType: msgCls}
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
func (w *worker) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.StreamModule).WithField("func", fmt.Sprintf("%T.%s", w, fn))
}
func NewWorker(tok string, sessCfg *tlg.SessionConfig, channelID int64, cacheRoot string) (IWorker, error) {
	cache := NewAccessHashCache(cacheRoot)
	docCache := NewDocCache(cacheRoot)
	w := worker{cl: tlg.NewTgClient(sessCfg, tok), channelID: channelID, cache: cache, docCache: docCache}
	if err := w.cl.Connect(); err != nil {
		return nil, fmt.Errorf("can not connect to worker: %w", err)
	}
	return &w, nil
}
