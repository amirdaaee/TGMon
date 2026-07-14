// Package bot provides handler logic for processing and forwarding media messages in the bot.
package bot

import (
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/ext"
	"go.uber.org/zap"
)

// IHandler defines the interface for bot handlers.
//
//go:generate mockgen -source=handler.go -destination=../../mocks/bot/handler.go -package=mocks_bot
type IHandler interface {
	// Register registers the handler with the given bot instance.
	Register(b *Bot)
}

func HandlerWithErrorMessage(fn handlers.CallbackResponse) handlers.CallbackResponse {
	ll := log.GetLogger(log.BotModule)
	_fn := func(c *ext.Context, u *ext.Update) error {
		err := fn(c, u)
		if err != nil {
			if _, err := c.Reply(u, ext.ReplyTextString(err.Error()), &ext.ReplyOpts{ReplyToMessageId: u.EffectiveMessage.ID}); err != nil {
				ll.With(zap.Error(err)).Error("error writing err message")
			}
			return err
		}
		return nil
	}
	return _fn
}
