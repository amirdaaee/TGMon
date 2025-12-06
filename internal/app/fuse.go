package app

import (
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/mazrean/kessoku"
)

//go:generate go tool kessoku $GOFILE
var _ = kessoku.Inject[*fuse.Server](
	"InitializeFuzeServer",
	kessoku.Provide(NewFuzeServer),
	kessoku.Provide(NewDbContainer),
	TgProviderSet,
)
