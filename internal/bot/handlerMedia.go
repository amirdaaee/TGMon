package bot

import (
	"fmt"

	ftypes "github.com/amirdaaee/TGMon/internal/facade/types"
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

// mediaHandler implements IHandler for processing media messages.
type mediaHandler struct {
	channelID       int64
	mediaFacade     ftypes.IFacade[types.MediaFileDoc]
	workerContainer stream.IWorkerPool
}

var _ IHandler = (*mediaHandler)(nil)

// Register registers the handler with the given bot instance and sets up message handlers.
func (h *mediaHandler) Register(b *Bot) {
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
func (h *mediaHandler) handleDoc(ctx *ext.Context, u *ext.Update) error {
	ll := h.getLogger("handleDoc")
	ll.Debug("new message received")
	if !u.EffectiveChat().IsAUser() {
		return NewBotError("message is not from a user", nil)
	}
	worker := h.workerContainer.GetNextWorker()
	if worker == nil {
		return NewBotError("no available worker", nil)
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
func (h *mediaHandler) buildMediaFileDoc(newDoc any, msgID int) (types.MediaFileDoc, error) {
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
func (h *mediaHandler) sendSuccessMsg(ctx *ext.Context, u *ext.Update, doc *types.MediaFileDoc) error {
	ll := h.getLogger("sendSuccessMsg")
	ll.Debugf("sending success message")
	m := fmt.Sprintf("ok: %s (%d)", doc.NameExt(), doc.Meta.FileID)
	if _, err := ctx.Reply(u, ext.ReplyTextString(m), &ext.ReplyOpts{ReplyToMessageId: u.EffectiveMessage.ID}); err != nil {
		return NewBotError("failed to send success message", err)
	}
	return nil
}

// getDispatcher returns the dispatcher from the given bot instance.
func (h *mediaHandler) getDispatcher(b *Bot) dispatcher.Dispatcher {
	return b.cl.GetClient().Dispatcher
}

// getLogger returns a logger entry with function context for the handler.
func (h *mediaHandler) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.BotModule).WithField("func", fmt.Sprintf("%T.%s", h, fn))
}

// NewMediaHandler creates a new handler instance with the given dependencies.
// Returns an error if any dependency is nil.
func NewMediaHandler(mediaFacade ftypes.IFacade[types.MediaFileDoc], channelID int64, wp stream.IWorkerPool) (IHandler, error) {
	if mediaFacade == nil {
		return nil, NewBotError("mediaFacade cannot be nil", nil)
	}
	if wp == nil {
		return nil, NewBotError("workerContainer cannot be nil", nil)
	}
	return &mediaHandler{
		mediaFacade:     mediaFacade,
		channelID:       channelID,
		workerContainer: wp,
	}, nil
}
