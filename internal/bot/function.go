// Package bot provides core bot logic, including message forwarding functionality.
package bot

import (
	"fmt"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/celestix/gotgproto/ext"
	tgTypes "github.com/celestix/gotgproto/types"
	"github.com/gotd/td/tg"
)

// forward forwards a message from one chat to another and returns the new message.
// It takes the context, update, and target chat ID, and returns the forwarded message or an error.
func forward(ctx *ext.Context, u *ext.Update, targetID int64) (*tgTypes.Message, error) {
	ll := log.GetLogger(log.BotModule).WithField("func", "forward")
	fromChat := u.EffectiveChat().GetID()
	toChat := targetID
	msgID := u.EffectiveMessage.ID
	ll.Debugf("forwarding message: %d -> %d : %d", fromChat, toChat, msgID)
	newUCls, err := ctx.ForwardMessages(fromChat, toChat, &tg.MessagesForwardMessagesRequest{
		ID: []int{u.EffectiveMessage.ID},
	})
	if err != nil {
		return nil, NewBotError("can not forward message", err)
	} else {
		ll.Debug("message forwarded")
	}
	// Type assertion: ensure newUCls is of type *tg.Updates
	upd, ok := newUCls.(*tg.Updates)
	if !ok {
		return nil, NewBotError(fmt.Sprintf("upd is not a *tg.Updates: %T", newUCls), nil)
	}
	var newMsg tg.MessageClass
	for c, u := range upd.Updates {
		ll.Debugf("update %d is %T", c, u)
		fwMsg, ok := u.(*tg.UpdateNewChannelMessage)
		if !ok {
			continue
		}
		newMsg = fwMsg.Message
		break
	}
	if newMsg == nil {
		return nil, NewBotError("no message in update found", nil)
	}
	m := tgTypes.ConstructMessage(newMsg)
	ll.Debugf("got forwarded message: %+v", m)
	return m, nil
}
