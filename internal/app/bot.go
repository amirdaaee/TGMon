package app

import (
	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/mazrean/kessoku"
)

//go:generate go tool kessoku $GOFILE
var _ = kessoku.Inject[*bot.Bot](
	"InitializeBot",
	kessoku.Provide(NewBot),
	kessoku.Provide(NewDbContainer),
	TgProviderSet,
	FacadeProviderSet,
)
