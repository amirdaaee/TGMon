package bot

import (
	"fmt"

	"github.com/amirdaaee/TGMon/internal/facade"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/stream"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/dispatcher/handlers/filters"
	"github.com/celestix/gotgproto/ext"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -source=handler.go -destination=../../mocks/bot/handler.go -package=mocks
type IHandler interface {
	Register(b *bot)
}
type handler struct {
	channelID       int64
	mediaFacade     facade.IFacade[types.MediaFileDoc]
	workerContainer stream.IWorkerContainer
}

func (h *handler) Register(b *bot) {
	ll := h.getLogger("Register")
	ll.Info("registering handler")
	dispatcher := h.getDispatcher(b)
	dispatcher.AddHandler(handlers.NewMessage(filters.Message.Media, h.handleDoc))
}
func (h *handler) handleDoc(ctx *ext.Context, u *ext.Update) error {
	ll := h.getLogger("handleDoc")
	ll.Debug("new message received")
	if !u.EffectiveChat().IsAUser() {
		return fmt.Errorf("message is not from a user")
	}
	worker := h.workerContainer.GetNextWorker()
	if _, err := worker.GetDoc(ctx, u.EffectiveMessage.GetID()); err != nil { // for validation only
		return fmt.Errorf("can not get document from user message: %w", err)
	}
	fwMsg, err := forward(ctx, u, h.channelID)
	if err != nil {
		return fmt.Errorf("can not forward message to channel: %w", err)
	}
	newDoc, err := worker.GetDoc(ctx, fwMsg.ID)
	if err != nil {
		return fmt.Errorf("can not get document from forwarded message: %w", err)
	}
	docMeta := types.MediaFileMeta{}
	if err := docMeta.FillFromDocument(newDoc); err != nil {
		return fmt.Errorf("can not fill document meta: %w", err)
	}
	docDoc := types.MediaFileDoc{
		Meta:      docMeta,
		MessageID: fwMsg.ID,
	}
	d, err := h.mediaFacade.CreateOne(ctx, &docDoc)
	if err != nil {
		return fmt.Errorf("can not create media file doc: %w", err)
	}
	ll.Infof("media file doc created: %v", docDoc)
	if err := h.sendSuccessMsg(ctx, u, d); err != nil {
		ll.WithError(err).Error("can not send success message")
	}
	return nil
}

func (h *handler) sendSuccessMsg(ctx *ext.Context, u *ext.Update, doc *types.MediaFileDoc) error {
	ll := h.getLogger("sendSuccessMsg")
	ll.Debugf("sending success message")
	m := fmt.Sprintf("ok: %s (%d)", doc.Meta.FileName, doc.Meta.FileID)
	if _, err := ctx.Reply(u, ext.ReplyTextString(m), &ext.ReplyOpts{ReplyToMessageId: u.EffectiveMessage.ID}); err != nil {
		return fmt.Errorf("failed to send success message: %w", err)
	}
	return nil
}
func (h *handler) getDispatcher(b *bot) dispatcher.Dispatcher {
	return b.cl.GetClient().Dispatcher
}

func (h *handler) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.BotModule).WithField("func", fmt.Sprintf("%T.%s", h, fn))
}

var _ IHandler = (*handler)(nil)

func NewHandler(mediaFacade facade.IFacade[types.MediaFileDoc], channelID int64, wp stream.IWorkerContainer) IHandler {
	return &handler{
		mediaFacade:     mediaFacade,
		channelID:       channelID,
		workerContainer: wp,
	}
}
