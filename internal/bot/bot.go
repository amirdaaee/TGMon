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
	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/ext"
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
type IClient interface {
	DeleteMessages(messageIDs []int) error
	UploadGetFile(ctx context.Context, request *tg.UploadGetFileRequest) (tg.UploadFileClass, error)
	ChannelsGetChannels(ctx context.Context, id []tg.InputChannelClass) (tg.MessagesChatsClass, error)
	ChannelsGetMessages(ctx context.Context, request *tg.ChannelsGetMessagesRequest) (tg.MessagesMessagesClass, error)
	Connect() error
	Idle() error
	GetDispatcher() dispatcher.Dispatcher
	GetName() string
	GetLogger() *logrus.Entry
	GetChannelID() int64
}

type TgClient struct {
	sessCfg         *SessionConfig
	token           string
	client          *gotgproto.Client
	targetChannelId int64
}

func (tg *TgClient) DeleteMessages(messageIDs []int) error {
	return tg.client.CreateContext().DeleteMessages(tg.GetChannelID(), messageIDs)
}
func (tg *TgClient) UploadGetFile(ctx context.Context, request *tg.UploadGetFileRequest) (tg.UploadFileClass, error) {
	return tg.client.API().UploadGetFile(ctx, request)
}
func (tg *TgClient) ChannelsGetChannels(ctx context.Context, id []tg.InputChannelClass) (tg.MessagesChatsClass, error) {
	return tg.client.API().ChannelsGetChannels(ctx, id)
}
func (tg *TgClient) ChannelsGetMessages(ctx context.Context, request *tg.ChannelsGetMessagesRequest) (tg.MessagesMessagesClass, error) {
	return tg.client.API().ChannelsGetMessages(ctx, request)
}
func (tg *TgClient) Connect() error {
	sessCfg := tg.sessCfg
	if tg.client != nil {
		tg.GetLogger().Warn("client is already connected")
		return nil
	}
	os.Mkdir(sessCfg.SessionDir, os.ModePerm)
	sessionDBPath := fmt.Sprintf("%s/worker-%s.sqlite3", sessCfg.SessionDir, strings.Split(tg.token, ":")[0])
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
		gotgproto.ClientTypeBot(tg.token),
		&clOpts,
	)
	if err != nil {
		return err
	}
	tg.client = client
	return nil
}
func (tg *TgClient) Idle() error {
	return tg.client.Idle()
}
func (tg *TgClient) GetDispatcher() dispatcher.Dispatcher {
	return tg.client.Dispatcher
}
func (tg *TgClient) GetName() string {
	return tg.client.Self.Username
}
func (tg *TgClient) GetLogger() *logrus.Entry {
	return logrus.WithField("bot", tg.GetName())
}
func (tg *TgClient) GetChannelID() int64 {
	return tg.targetChannelId
}

type tgClientFactory func(token string, sessConfig *SessionConfig) IClient

func NewTgClient(token string, sessConfig *SessionConfig) IClient {
	return &TgClient{
		sessCfg: sessConfig,
		token:   token,
	}
}

// ...
type Worker struct {
	cl               IClient
	inputChannel     tg.InputChannelClass
	inputChannelLock sync.Mutex
	accCache         *lru.Cache[int64, int64]
	accCacheLock     sync.Mutex
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
func (w *Worker) GetChannelMessages(ctx context.Context, msgID []int) (*tg.MessagesChannelMessages, error) {
	channel, err := w.GetChannel(ctx)
	if err != nil {
		return nil, fmt.Errorf("can not get channel while getting messages: %s", err)
	}
	return w.getChannelMessages(ctx, channel, msgID)
}
func (w *Worker) DeleteMessages(msgID []int) error {
	return w.cl.DeleteMessages(msgID)
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
	res, err := w.cl.UploadGetFile(ctx, req)
	if err != nil {
		return nil, err
	}
	thumbFile := res.(*tg.UploadFile)
	return thumbFile.Bytes, nil
}
func (w *Worker) getLogger() *logrus.Entry {
	botUsername := "?"
	if w.cl != nil {
		botUsername = w.cl.GetName()
	}
	return logrus.WithField("worker", botUsername)
}
func (w *Worker) getInputChannel(ctx context.Context) (tg.InputChannelClass, error) {
	targetChannelId := w.cl.GetChannelID()
	chatList, err := w.cl.ChannelsGetChannels(ctx, []tg.InputChannelClass{&tg.InputChannel{ChannelID: targetChannelId}})
	if err != nil {
		return nil, fmt.Errorf("can not get channel: %s", err)
	}
	var channel tg.InputChannelClass
	for _, cht := range chatList.GetChats() {
		if cht.GetID() == targetChannelId {
			if chn, ok := cht.(*tg.Channel); !ok {
				return nil, fmt.Errorf("target channel is not a channel: %T", cht)
			} else {
				channel = chn.AsInput()
				break
			}
		} else {
			return nil, fmt.Errorf("chat id mismatch: %d vs %d", cht.GetID(), targetChannelId)
		}
	}
	return channel, nil
}
func (w *Worker) getChannelMessages(ctx context.Context, channel tg.InputChannelClass, msgID []int) (*tg.MessagesChannelMessages, error) {
	inputMsgList := []tg.InputMessageClass{}
	for _, id := range msgID {
		inputMsgList = append(inputMsgList, &tg.InputMessageID{ID: id})
	}
	allMsgsCls, err := w.cl.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{Channel: channel, ID: inputMsgList})
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
	msg, err := w.GetChannelMessages(ctx, []int{doc.MessageID})
	if err != nil {
		return 0, fmt.Errorf("error getting message of document: %s", err)
	}
	newDoc := TelegramDocument{}
	if err := newDoc.FromMessage(msg.Messages[0]); err != nil {
		return 0, fmt.Errorf("error getting document of message of document: %s", err)
	}
	return newDoc.AccessHash, nil
}
func NewWorker(client IClient) (*Worker, error) {
	accCache, err := lru.New[int64, int64](128)
	if err != nil {
		return nil, fmt.Errorf("can not create worker accCache: %s", err)
	}
	w := &Worker{accCache: accCache, cl: client}
	return w, nil
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
func NewWorkerPool(tokens []string, sessCfg *SessionConfig, clientFactory tgClientFactory) (*WorkerPool, error) {
	if clientFactory == nil {
		clientFactory = NewTgClient
	}
	wp := WorkerPool{}
	var wg sync.WaitGroup
	for _, tok := range tokens {
		wg.Add(1)
		go func(_i string) {
			defer wg.Done()
			ll := logrus.WithField("worker", _i)
			ll.Info("initiating worker")
			cl := clientFactory(_i, sessCfg)
			w, err := NewWorker(cl)
			if err != nil {
				ll.WithError(err).Warn("can not create worker")
				return
			}
			if err := w.cl.Connect(); err != nil {
				ll.WithError(err).Warn("can not initiate worker")
				return
			}
			wp.mut.Lock()
			defer wp.mut.Unlock()
			wp.Bots = append(wp.Bots, w)
			ll.Info("worker initaited")
		}(tok)
	}
	wg.Wait()
	if len(wp.Bots) == 0 {
		return nil, fmt.Errorf("no worker is avaiable")
	}
	return &wp, nil
}

// ...
type Master struct {
	Bot         *Worker
	mediaFacade facade.IMediaFacade
	mongo       db.IMongo
}

func (mstr *Master) Start() error {
	return mstr.Bot.cl.Idle()
}
func (mstr *Master) getLogger() *logrus.Entry {
	mstrUsername := "?"
	if mstr.Bot.cl != nil {
		mstrUsername = mstr.Bot.cl.GetName()
	}
	return logrus.WithField("master", mstrUsername)
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
	if chatId != mstr.Bot.cl.GetChannelID() {
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
func NewMaster(token string, sessCfg *SessionConfig, facade *facade.MediaFacade, clientFactory tgClientFactory) (*Master, error) {
	if clientFactory == nil {
		clientFactory = NewTgClient
	}
	cl := clientFactory(token, sessCfg)
	w, err := NewWorker(cl)
	if err != nil {
		return nil, fmt.Errorf("can not create client: %s", err)
	}
	if err := w.cl.Connect(); err != nil {
		return nil, fmt.Errorf("error initating master bot: %s", err)
	}
	mstr := &Master{Bot: w, mediaFacade: facade}
	dispatch := w.cl.GetDispatcher()
	dispatch.AddHandler(
		handlers.NewMessage(nil, mstr.handle),
	)
	return mstr, nil
}
