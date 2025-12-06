package app

import (
	"net/http"

	"github.com/mazrean/kessoku"
)

//go:generate go tool kessoku $GOFILE
var _ = kessoku.Inject[*http.Server](
	"InitializeWebServer",
	kessoku.Provide(NewWebServer),
	kessoku.Provide(NewWebHandler),
	kessoku.Provide(NewGinEngine),
	kessoku.Provide(NewDbContainer),
	kessoku.Provide(NewStashQlClient),
	FacadeProviderSet,
	TgProviderSet,
)
