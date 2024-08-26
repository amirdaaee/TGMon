package bot

import (
	"fmt"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/sirupsen/logrus"
)

type Notifier struct {
	DocNotifier *docNotifier
}
type docNotifier struct {
	channelId int64
	Chan      chan *Document
}

func (dn *docNotifier) handle(ctx *ext.Context, u *ext.Update) error {
	chatId := u.EffectiveChat().GetID()
	effMsg := u.EffectiveMessage
	ll := logrus.WithField("chat-id", chatId).WithField("message-id", effMsg.ID)
	ll.Debug("new message")
	if chatId != dn.channelId {
		ll.Debug("message not in channel")
		return dispatcher.EndGroups
	}
	supported, err := SupportedMediaFilter(effMsg)
	if err != nil && err != dispatcher.EndGroups {
		return fmt.Errorf("can not filter supported media type: %s", err)
	}
	if !supported {
		ll.Debug("message not supported")
		return dispatcher.EndGroups
	}
	doc := Document{}
	if err := doc.FromMessage(effMsg.Message); err != nil {
		return fmt.Errorf("error getting document of message %s", err)
	}
	dn.Chan <- &doc
	return nil
}
