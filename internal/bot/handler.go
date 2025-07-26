// Package bot provides handler logic for processing and forwarding media messages in the bot.
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
	"github.com/gotd/td/tg"
	"github.com/sirupsen/logrus"
)

// IHandler defines the interface for bot handlers.
//
//go:generate mockgen -source=handler.go -destination=../../mocks/bot/handler.go -package=mocks
type IHandler interface {
	// Register registers the handler with the given bot instance.
	Register(b *Bot)
}

// handler implements IHandler for processing media messages.
type handler struct {
	channelID       int64
	mediaFacade     facade.IFacade[types.MediaFileDoc]
	workerContainer stream.IWorkerContainer
}

// Register registers the handler with the given bot instance and sets up message handlers.
func (h *handler) Register(b *Bot) {
	ll := h.getLogger("Register")
	ll.Info("registering handler")
	if b == nil {
		ll.Error("bot instance is nil in Register")
		return
	}
	dispatcher := h.getDispatcher(b)
	dispatcher.AddHandler(handlers.NewMessage(filters.Message.Media, HandlerWithErrorMessage(h.handleDoc)))
}

// handleDoc processes incoming media messages from users, forwards them, and stores metadata.
func (h *handler) handleDoc(ctx *ext.Context, u *ext.Update) error {
	ll := h.getLogger("handleDoc")
	ll.Debug("new message received")
	if !u.EffectiveChat().IsAUser() {
		return NewBotError("message is not from a user", nil)
	}
	worker := h.workerContainer.GetNextWorker()
	if worker == nil {
		return NewBotError("no available worker", nil)
	}
	// Validate original document
	if _, err := worker.GetDoc(ctx, u.EffectiveMessage.GetID()); err != nil {
		return NewBotError("can not get document from user message", err)
	}
	// Forward message and process result
	fwMsg, err := forward(ctx, u, h.channelID)
	if err != nil {
		return NewBotError("can not forward message to channel", err)
	}
	// Get document from forwarded message
	newDoc, err := worker.GetDoc(ctx, fwMsg.ID)
	if err != nil {
		return NewBotError("can not get document from forwarded message", err)
	}
	// Build and store document metadata
	docDoc, err := h.buildMediaFileDoc(newDoc, fwMsg.ID)
	if err != nil {
		return NewBotError("can not build media file doc", err)
	}
	d, err := h.mediaFacade.CreateOne(ctx, &docDoc)
	if err != nil {
		return NewBotError("can not create media file doc", err)
	}
	ll.Infof("media file doc created: %v", docDoc)
	if err := h.sendSuccessMsg(ctx, u, d); err != nil {
		ll.WithError(err).Error("can not send success message")
	}
	return nil
}

// buildMediaFileDoc creates a MediaFileDoc from a document and message ID.
func (h *handler) buildMediaFileDoc(newDoc any, msgID int) (types.MediaFileDoc, error) {
	docMeta := types.MediaFileMeta{}
	doc, ok := newDoc.(*tg.Document)
	if !ok {
		return types.MediaFileDoc{}, NewBotError(fmt.Sprintf("newDoc is not a *tg.Document: %T", newDoc), nil)
	}
	if err := docMeta.FillFromDocument(doc); err != nil {
		return types.MediaFileDoc{}, NewBotError("can not fill document meta", err)
	}
	return types.MediaFileDoc{
		Meta:      docMeta,
		MessageID: msgID,
	}, nil
}

// sendSuccessMsg sends a confirmation message to the user after successful processing.
func (h *handler) sendSuccessMsg(ctx *ext.Context, u *ext.Update, doc *types.MediaFileDoc) error {
	ll := h.getLogger("sendSuccessMsg")
	ll.Debugf("sending success message")
	m := fmt.Sprintf("ok: %s (%d)", doc.Meta.FileName, doc.Meta.FileID)
	if _, err := ctx.Reply(u, ext.ReplyTextString(m), &ext.ReplyOpts{ReplyToMessageId: u.EffectiveMessage.ID}); err != nil {
		return NewBotError("failed to send success message", err)
	}
	return nil
}

// getDispatcher returns the dispatcher from the given bot instance.
func (h *handler) getDispatcher(b *Bot) dispatcher.Dispatcher {
	return b.cl.GetClient().Dispatcher
}

// getLogger returns a logger entry with function context for the handler.
func (h *handler) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.BotModule).WithField("func", fmt.Sprintf("%T.%s", h, fn))
}

var _ IHandler = (*handler)(nil)

// NewHandler creates a new handler instance with the given dependencies.
// Returns an error if any dependency is nil.
func NewHandler(mediaFacade facade.IFacade[types.MediaFileDoc], channelID int64, wp stream.IWorkerContainer) (IHandler, error) {
	if mediaFacade == nil {
		return nil, NewBotError("mediaFacade cannot be nil", nil)
	}
	if wp == nil {
		return nil, NewBotError("workerContainer cannot be nil", nil)
	}
	return &handler{
		mediaFacade:     mediaFacade,
		channelID:       channelID,
		workerContainer: wp,
	}, nil
}

// ---
func HandlerWithErrorMessage(fn handlers.CallbackResponse) handlers.CallbackResponse {
	ll := log.GetLogger(log.BotModule)
	_fn := func(c *ext.Context, u *ext.Update) error {
		err := fn(c, u)
		if err != nil {
			if _, err := c.Reply(u, ext.ReplyTextString(err.Error()), &ext.ReplyOpts{ReplyToMessageId: u.EffectiveMessage.ID}); err != nil {
				ll.WithError(err).Error("error writing err message")
			}
			return err
		}
		return nil
	}
	return _fn
}
