package bot

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/sessionMaker"
	"github.com/glebarez/sqlite"
	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/contrib/middleware/ratelimit"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/tg"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
	"golang.org/x/time/rate"
)

// ...
type Worker struct {
	Token           string
	Client          *gotgproto.Client
	TargetChannelId int64
}

func (w *Worker) String() string {
	return fmt.Sprintf("{Worker (%s|@%s)}", w.Token, w.Client.Self.Username)
}
func (w *Worker) DeleteMessages(msgID []int) error {
	return w.Client.CreateContext().DeleteMessages(w.TargetChannelId, msgID)
}
func (w *Worker) GetThumbnail(doc *Document, ctx context.Context) ([]byte, error) {
	thmb := doc.Thumbs[0].(*tg.PhotoSize)
	size := thmb.Type
	loc_ := tg.InputDocumentFileLocation{}
	loc_.FillFrom(doc.AsInputDocumentFileLocation())
	loc_.ThumbSize = size
	req := &tg.UploadGetFileRequest{
		Location: &loc_,
		Limit:    1024 * 1024,
		Precise:  false,
	}
	res, err := w.Client.API().UploadGetFile(ctx, req)
	if err != nil {
		return nil, err
	}
	thumbFile := res.(*tg.UploadFile)
	return thumbFile.Bytes, nil
}

// ...
type WorkerPool struct {
	Bots     []*Worker
	curIndex int
	mut      sync.Mutex
}

func (wp *WorkerPool) GetNextWorker() *Worker {
	wp.mut.Lock()
	defer wp.mut.Unlock()
	index := (wp.curIndex + 1) % len(wp.Bots)
	wp.curIndex = index
	worker := wp.Bots[index]
	logrus.Debugf("Using worker %s", worker.String())
	return worker
}

// ...
type SessionConfig struct {
	SocksProxy string
	SessionDir string
	AppID      int
	AppHash    string
	ChannelId  int64
}

// ...
func NewMaster(token string, sessCfg *SessionConfig) (*Notifier, error) {
	client, err := startClient(token, sessCfg)
	if err != nil {
		return nil, fmt.Errorf("error starting master bot: %s", err)
	}
	notifier := Notifier{
		DocNotifier: &docNotifier{channelId: sessCfg.ChannelId, Chan: make(chan *Document)},
	}
	client.Dispatcher.AddHandler(
		handlers.NewMessage(nil, notifier.DocNotifier.handle),
	)
	go func() {
		if err := client.Idle(); err != nil {
			logrus.WithError(err).Fatal("error idle bot")
		}
	}()
	return &notifier, nil
}

func NewWorkerPool(tokens []string, sessCfg *SessionConfig) (*WorkerPool, error) {
	var wg sync.WaitGroup
	wp := WorkerPool{}
	for _, tok := range tokens {
		wg.Add(1)
		go func(_i string) {
			defer wg.Done()
			ll := logrus.WithField("worker", _i)
			client, err := startClient(_i, sessCfg)
			if err != nil {
				ll.WithError(err).Warn("can not start worker")
				return
			}
			wp.mut.Lock()
			defer wp.mut.Unlock()
			wp.Bots = append(wp.Bots, &Worker{Client: client, Token: _i, TargetChannelId: sessCfg.ChannelId})
			ll.Info("worker started")
		}(tok)
	}
	wg.Wait()
	if len(wp.Bots) == 0 {
		return nil, fmt.Errorf("no worker is avaiable")
	}
	return &wp, nil
}

func startClient(botToken string, sessCfg *SessionConfig) (*gotgproto.Client, error) {
	os.Mkdir(sessCfg.SessionDir, os.ModePerm)
	sessionType := sessionMaker.SqlSession(sqlite.Open(fmt.Sprintf("%s/worker-%s.sqlite3", sessCfg.SessionDir, strings.Split(botToken, ":")[0])))
	clOpts := gotgproto.ClientOpts{
		Session:          sessionType,
		DisableCopyright: true,
		Middlewares: []telegram.Middleware{
			floodwait.NewSimpleWaiter().WithMaxRetries(10).WithMaxWait(5 * time.Second),
			ratelimit.New(rate.Every(time.Millisecond*100), 5),
		},
	}
	if resolver, err := getSocksDialer(sessCfg); err != nil {
		logrus.WithError(err).Error("can not get socks dialer. using default")
	} else if resolver != nil {
		clOpts.Resolver = *resolver
	}
	client, err := gotgproto.NewClient(
		sessCfg.AppID,
		sessCfg.AppHash,
		gotgproto.ClientTypeBot(botToken),
		&clOpts,
	)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func getSocksDialer(sessCfg *SessionConfig) (*dcs.Resolver, error) {
	proxyUriStr := sessCfg.SocksProxy
	if proxyUriStr == "" {
		return nil, nil
	}
	proxyUri, err := url.Parse(string(proxyUriStr))
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
