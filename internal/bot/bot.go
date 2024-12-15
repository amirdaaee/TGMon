package bot

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/facade"
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/ext"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/sessionMaker"
	"github.com/glebarez/sqlite"
	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/contrib/middleware/ratelimit"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/tg"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
	"golang.org/x/time/rate"
)

// ...
type Worker struct {
	token            string
	client           *gotgproto.Client
	targetChannelId  int64
	inputChannel     tg.InputChannelClass
	inputChannelLock sync.Mutex
	accCache         *lru.Cache[int64, int64]
	accCacheLock     sync.Mutex
	sessCfg          *SessionConfig
}

func (w *Worker) GetChannel(ctx context.Context) (tg.InputChannelClass, error) {
	w.inputChannelLock.Lock()
	defer w.inputChannelLock.Unlock()
	if w.inputChannel == nil {
		ch, err := w.getInputChannel(ctx)
		if err != nil {
			return nil, err
		}
		w.inputChannel = ch
	}
	return w.inputChannel, nil
}
func (w *Worker) GetMessages(ctx context.Context, msgID []int) (*tg.MessagesChannelMessages, error) {
	channel, err := w.GetChannel(ctx)
	if err != nil {
		return nil, fmt.Errorf("can not get channel while getting messages: %s", err)
	}
	return w.getChannelMessages(ctx, channel, msgID)
}
func (w *Worker) GetDocAccHash(doc *TelegramDocument, ctx context.Context) (int64, error) {
	w.accCacheLock.Lock()
	defer w.accCacheLock.Unlock()
	accHash, ok := w.accCache.Get(doc.ID)
	if !ok {
		accHash, err := w.getDocAccHash(ctx, doc)
		if err != nil {
			return 0, err
		}
		w.accCache.Add(doc.ID, accHash)
		return accHash, nil
	}
	return accHash, nil
}
func (w *Worker) DeleteMessages(msgID []int) error {
	return w.client.CreateContext().DeleteMessages(w.targetChannelId, msgID)
}
func (w *Worker) GetThumbnail(doc *TelegramDocument, ctx context.Context) ([]byte, error) {
	thmb := doc.Thumbs[0].(*tg.PhotoSize)
	size := thmb.Type
	loc_ := tg.InputDocumentFileLocation{}
	_, err := w.GetDocAccHash(doc, ctx)
	if err != nil {
		return nil, fmt.Errorf("error updating access hash: %s", err)
	}
	loc_.FillFrom(doc.AsInputDocumentFileLocation())
	loc_.ThumbSize = size
	req := &tg.UploadGetFileRequest{
		Location: &loc_,
		Limit:    1024 * 1024,
		Precise:  false,
	}
	res, err := w.client.API().UploadGetFile(ctx, req)
	if err != nil {
		return nil, err
	}
	thumbFile := res.(*tg.UploadFile)
	return thumbFile.Bytes, nil
}
func (w *Worker) getLogger() *logrus.Entry {
	botUsername := "?"
	if w.client != nil {
		botUsername = w.client.Self.Username
	}
	s := fmt.Sprintf("{Worker (%s|@%s)}", w.token, botUsername)
	return logrus.WithField("worker", s)
}
func (w *Worker) getInputChannel(ctx context.Context) (tg.InputChannelClass, error) {
	chatList, err := w.client.API().ChannelsGetChannels(ctx, []tg.InputChannelClass{&tg.InputChannel{ChannelID: w.targetChannelId}})
	if err != nil {
		return nil, fmt.Errorf("can not get channel")
	}
	var channel tg.InputChannelClass
	for _, cht := range chatList.GetChats() {
		if cht.GetID() == w.targetChannelId {
			if chn, ok := cht.(*tg.Channel); !ok {
				return nil, fmt.Errorf("target channel is not a channel")
			} else {
				channel = chn.AsInput()
				break
			}
		}
	}
	if channel == nil {
		return nil, fmt.Errorf("target channel not found")
	}
	return channel, nil
}
func (w *Worker) getChannelMessages(ctx context.Context, channel tg.InputChannelClass, msgID []int) (*tg.MessagesChannelMessages, error) {
	inputMsgList := []tg.InputMessageClass{}
	for _, id := range msgID {
		inputMsgList = append(inputMsgList, &tg.InputMessageID{ID: id})
	}
	allMsgsCls, err := w.client.API().ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{Channel: channel, ID: inputMsgList})
	if err != nil {
		return nil, fmt.Errorf("can not get messages of channel: %s", err)
	}
	allMsgs, ok := allMsgsCls.(*tg.MessagesChannelMessages)
	if !ok {
		return nil, fmt.Errorf("class of messages is %T, not MessagesChannelMessages", allMsgsCls)
	}
	return allMsgs, nil
}
func (w *Worker) getDocAccHash(ctx context.Context, doc *TelegramDocument) (int64, error) {
	msg, err := w.GetMessages(ctx, []int{doc.MessageID})
	if err != nil {
		return 0, fmt.Errorf("error getting message of document: %s", err)
	}
	newDoc := TelegramDocument{}
	if err := newDoc.FromMessage(msg.Messages[0]); err != nil {
		return 0, fmt.Errorf("error getting document of message of document: %s", err)
	}
	return newDoc.AccessHash, nil
}
func (w *Worker) startClient() error {
	sessCfg := w.sessCfg
	if w.client != nil {
		w.getLogger().Warn("client is already started")
		return nil
	}
	os.Mkdir(sessCfg.SessionDir, os.ModePerm)
	sessionDBPath := fmt.Sprintf("%s/worker-%s.sqlite3", sessCfg.SessionDir, strings.Split(w.token, ":")[0])
	sessionType := sessionMaker.SqlSession(sqlite.Open(sessionDBPath))
	clOpts := gotgproto.ClientOpts{
		Session:          sessionType,
		DisableCopyright: true,
		Middlewares: []telegram.Middleware{
			floodwait.NewSimpleWaiter().WithMaxRetries(10).WithMaxWait(5 * time.Second),
			ratelimit.New(rate.Every(time.Millisecond*100), 5),
		},
	}
	if resolver, err := sessCfg.getSocksDialer(); err != nil {
		logrus.WithError(err).Error("can not get socks dialer. using default")
	} else if resolver != nil {
		clOpts.Resolver = *resolver
	}
	client, err := gotgproto.NewClient(
		sessCfg.AppID,
		sessCfg.AppHash,
		gotgproto.ClientTypeBot(w.token),
		&clOpts,
	)
	if err != nil {
		return err
	}
	w.client = client
	return nil
}

// ...
type WorkerPool struct {
	Bots     []*Worker
	curIndex int
	mut      sync.Mutex
}

func (wp *WorkerPool) SelectNextWorker() *Worker {
	wp.mut.Lock()
	defer wp.mut.Unlock()
	index := (wp.curIndex + 1) % len(wp.Bots)
	wp.curIndex = index
	worker := wp.Bots[index]
	worker.getLogger().Debugf("using this worker (%d/%d)", index+1, len(wp.Bots))
	return worker
}

// ...
type Master struct {
	Bot         *Worker
	mediaFacade facade.IMediaFacade
	mongo       db.IMongo
}

func (mstr *Master) Start() error {
	return mstr.Bot.client.Idle()
}
func (mstr *Master) getLogger() *logrus.Entry {
	mstrUsername := "?"
	if mstr.Bot.client != nil {
		mstrUsername = mstr.Bot.client.Self.Username
	}
	s := fmt.Sprintf("{Master (%s|@%s)}", mstr.Bot.token, mstrUsername)
	return logrus.WithField("worker", s)
}
func (mstr *Master) handle(ctx *ext.Context, u *ext.Update) error {
	effMsg := u.EffectiveMessage
	ll := mstr.getLogger()
	ll.Debug("new message")
	if supported := mstr.filter(u, ll); !supported {
		ll.Debug("stopped further processing")
		return dispatcher.EndGroups
	}
	doc := TelegramDocument{}
	if err := doc.FromMessage(effMsg.Message); err != nil {
		return fmt.Errorf("error getting document of message %s", err)
	}
	// ...
	cl, err := mstr.mongo.GetClient()
	if err != nil {
		return fmt.Errorf("can not get mongo client %s", err)
	}
	defer cl.Disconnect(ctx)
	// ...
	mediaDoc := mstr.dbDocFromMessage(&doc)
	thumb, err := mstr.Bot.GetThumbnail(&doc, ctx)
	if err != nil {
		ll.WithError(err).Error("can not get thumbnail")
	}
	media := facade.NewFullMediaData(mediaDoc, thumb)
	if _, err := mstr.mediaFacade.Create(ctx, media, cl); err != nil {
		return fmt.Errorf("error in facade create: %s", err)
	}
	ll.Debug("new media added")
	return nil
}
func (mstr *Master) dbDocFromMessage(doc *TelegramDocument) *db.MediaFileDoc {
	docMeta := doc.GetMetadata()
	mediaDoc := db.MediaFileDoc{
		Location:  docMeta.Location,
		FileSize:  docMeta.FileSize,
		FileName:  docMeta.FileName,
		MimeType:  docMeta.MimeType,
		FileID:    docMeta.DocID,
		MessageID: doc.MessageID,
		Duration:  docMeta.Duration,
		DateAdded: time.Now().Unix(),
	}
	return &mediaDoc
}
func (mstr *Master) filter(u *ext.Update, ll *logrus.Entry) bool {
	ll = ll.WithField("at", "filter")
	chatId := u.EffectiveChat().GetID()
	effMsg := u.EffectiveMessage
	if chatId != mstr.Bot.targetChannelId {
		ll.Debug("message not in channel")
		return false
	}
	if effMsg.Media == nil {
		ll.Debug("message doesn't contain media")
		return false
	}
	switch m := effMsg.Media.(type) {
	case *tg.MessageMediaDocument:
		return true
	default:
		ll.Debugf("message type (%T) not supported", m)
		return false
	}
}

// ...
type SessionConfig struct {
	SocksProxy string
	SessionDir string
	AppID      int
	AppHash    string
	ChannelId  int64
}

func (sessCfg *SessionConfig) getSocksDialer() (*dcs.Resolver, error) {
	proxyUriStr := sessCfg.SocksProxy
	if proxyUriStr == "" {
		return nil, nil
	}
	proxyUri, err := url.Parse(proxyUriStr)
	if err != nil {
		return nil, fmt.Errorf("can not parse proxy url (%s): %s", proxyUriStr, err)
	}
	uPass, _ := proxyUri.User.Password()
	sock5, err := proxy.SOCKS5("tcp", proxyUri.Host, &proxy.Auth{
		User:     proxyUri.User.Username(),
		Password: uPass,
	}, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("can not create socks proxy (%s): %s", proxyUriStr, err)
	}
	dc := sock5.(proxy.ContextDialer)
	dialler := dcs.Plain(dcs.PlainOptions{
		Dial: dc.DialContext,
	})
	return &dialler, nil
}

// ...
func NewWorker(token string, sessCfg *SessionConfig) (*Worker, error) {
	accCache, err := lru.New[int64, int64](128)
	if err != nil {
		return nil, fmt.Errorf("can not create worker accCache: %s", err)
	}
	w := &Worker{token: token, targetChannelId: sessCfg.ChannelId, accCache: accCache, sessCfg: sessCfg}
	return w, nil
}
func NewWorkerPool(tokens []string, sessCfg *SessionConfig) (*WorkerPool, error) {
	var wg sync.WaitGroup
	wp := WorkerPool{}
	for _, tok := range tokens {
		wg.Add(1)
		go func(_i string) {
			defer wg.Done()
			ll := logrus.WithField("worker", _i)
			w, err := NewWorker(_i, sessCfg)
			if err != nil {
				ll.WithError(err).Warn("can not create worker")
				return
			}
			if err := w.startClient(); err != nil {
				ll.WithError(err).Warn("can not start worker")
				return
			}
			wp.mut.Lock()
			defer wp.mut.Unlock()
			wp.Bots = append(wp.Bots, w)
			ll.Info("worker started")
		}(tok)
	}
	wg.Wait()
	if len(wp.Bots) == 0 {
		return nil, fmt.Errorf("no worker is avaiable")
	}
	return &wp, nil
}
func NewMaster(token string, sessCfg *SessionConfig, facade *facade.MediaFacade) (*Master, error) {
	w, err := NewWorker(token, sessCfg)
	if err != nil {
		return nil, fmt.Errorf("can not create client: %s", err)
	}
	if err := w.startClient(); err != nil {
		return nil, fmt.Errorf("error starting master bot: %s", err)
	}
	mstr := &Master{Bot: w, mediaFacade: facade}
	w.client.Dispatcher.AddHandler(
		handlers.NewMessage(nil, mstr.handle),
	)
	return mstr, nil
}
