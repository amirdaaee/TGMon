package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/amirdaaee/TGMon/config"
	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/sessionMaker"
	"github.com/glebarez/sqlite"
	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/contrib/middleware/ratelimit"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/dcs"
	"go.uber.org/zap"
	"golang.org/x/net/proxy"
	"golang.org/x/time/rate"
)

func StartMainBot(log *zap.Logger) (*gotgproto.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	resultChan := make(chan struct {
		client *gotgproto.Client
		err    error
	})
	go func(ctx context.Context) {
		client, err := startClient(log, config.Config().BotToken, -1)
		resultChan <- struct {
			client *gotgproto.Client
			err    error
		}{client, err}
	}(ctx)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultChan:
		if result.err != nil {
			return nil, result.err
		}
		addDispatcher(log, result.client)
		return result.client, nil
	}
}

// ...
func addDispatcher(log *zap.Logger, client *gotgproto.Client) {
	log.Debug("add dispatcher to bot")
	client.Dispatcher.AddHandler(
		handlers.NewMessage(nil, addMediaDB),
	)
}

func startClient(l *zap.Logger, botToken string, index int) (*gotgproto.Client, error) {
	log := l.Named("Worker").Sugar()
	log.Infof("Starting worker with index - %d", index)
	sessionType := sessionMaker.SqlSession(sqlite.Open(fmt.Sprintf("%s/worker-%d", config.Config().SessionDir, index)))
	clOpts := gotgproto.ClientOpts{
		Session:          sessionType,
		DisableCopyright: true,
		Middlewares: []telegram.Middleware{
			floodwait.NewSimpleWaiter().WithMaxRetries(10).WithMaxWait(5 * time.Second),
			ratelimit.New(rate.Every(time.Millisecond*100), 5),
		},
	}
	if resolver := getSocksDialler(); resolver != nil {
		clOpts.Resolver = *resolver
	}
	client, err := gotgproto.NewClient(
		config.Config().AppID,
		config.Config().AppHash,
		gotgproto.ClientTypeBot(botToken),
		&clOpts,
	)
	if err != nil {
		return nil, err
	}
	return client, nil

}

func getSocksDialler() *dcs.Resolver {
	proxyUri := config.Config().TGSocksProxy
	if proxyUri.Scheme == "" {
		return nil
	}
	uPass, _ := proxyUri.User.Password()
	sock5, _ := proxy.SOCKS5("tcp", proxyUri.Host, &proxy.Auth{
		User:     proxyUri.User.Username(),
		Password: uPass,
	}, proxy.Direct)
	dc := sock5.(proxy.ContextDialer)
	dialler := dcs.Plain(dcs.PlainOptions{
		Dial: dc.DialContext,
	})
	return &dialler
}
