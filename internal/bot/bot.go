package bot

import (
	"fmt"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/tlg"
	"github.com/sirupsen/logrus"
)

type Bot struct {
	cl tlg.IClient
}

func (b *Bot) Start() error {
	ll := b.getLogger("Start")
	ll.Info("starting client")
	return b.cl.GetClient().Idle()
}

func (b *Bot) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.BotModule).WithField("func", fmt.Sprintf("%T.%s", b, fn))
}
func NewBot(cl tlg.IClient) (*Bot, error) {
	err := cl.Connect()
	if err != nil {
		return nil, fmt.Errorf("can not connect to bot: %w", err)
	}
	b := Bot{cl: cl}
	return &b, nil
}
