package tlg

import (
	"fmt"
	"net/url"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/gotd/td/telegram/dcs"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
)

type SessionConfig struct {
	SocksProxy string
	SessionDir string
	AppID      int
	AppHash    string
	ChannelId  int64
}

func (sessCfg *SessionConfig) getSocksDialer() (*dcs.Resolver, error) {
	ll := sessCfg.getLogger("getSocksDialer")
	proxyUriStr := sessCfg.SocksProxy
	if proxyUriStr == "" {
		ll.Info("no socks proxy provided")
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
	ll.Info("socks dialer created")
	return &dialler, nil
}

func (sessCfg *SessionConfig) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.TlgModule).WithField("func", fmt.Sprintf("%T.%s", sessCfg, fn))
}
