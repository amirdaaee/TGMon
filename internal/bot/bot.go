// Package bot provides the core bot logic and integration with the Telegram client.
package bot

import (
	"fmt"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/tlg"
	"github.com/sirupsen/logrus"
)

// Bot represents the core bot instance that manages the Telegram client.
type Bot struct {
	cl tlg.IClient
}

// Start starts the bot client and blocks until stopped.
// It returns an error if the client fails to idle.
func (b *Bot) Start() error {
	ll := b.getLogger("Start")
	ll.Info("starting client")
	return b.cl.GetClient().Idle()
}

// getLogger returns a logger entry with function context for the Bot.
func (b *Bot) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.BotModule).WithField("func", fmt.Sprintf("%T.%s", b, fn))
}

// NewBot creates and connects a new Bot instance with the given Telegram client.
// Returns an error if the client is nil or fails to connect.
func NewBot(cl tlg.IClient) (*Bot, error) {
	if cl == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}
	err := cl.Connect()
	if err != nil {
		return nil, fmt.Errorf("can not connect to bot: %w", err)
	}
	b := Bot{cl: cl}
	return &b, nil
}
