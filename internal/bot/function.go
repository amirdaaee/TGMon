package bot

import (
	"fmt"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/celestix/gotgproto/ext"
	tgTypes "github.com/celestix/gotgproto/types"
	"github.com/gotd/td/tg"
)

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
		return nil, fmt.Errorf("can not forward message: %w", err)
	} else {
		ll.Debug("message forwarded")
	}
	upd, ok := newUCls.(*tg.Updates)
	if !ok {
		return nil, fmt.Errorf("upd is not a *tg.Updates: %T", newUCls)
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
		return nil, fmt.Errorf("no message in update found")
	}
	m := tgTypes.ConstructMessage(newMsg)
	ll.Debugf("got forwarded message: %+v", m)
	return m, nil
}
