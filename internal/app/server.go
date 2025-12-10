//go:build wireinject
// +build wireinject

package app

import (
	"context"
	"net/http"

	"github.com/amirdaaee/TGMon/internal/config"
	"github.com/google/wire"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type ServerModules struct {
	WebServer  *http.Server
	FuzeServer *fuse.Server
}

func InitializeServer(ctx context.Context, cfg *config.ConfigType) (*ServerModules, error) {
	wire.Build(WebHandlerProviderSet, NewStashQlClient, FuseProviderSet, NewDbContainer, TgProviderSet, FacadeProviderSet, wire.Struct(new(ServerModules), "*"))
	return nil, nil
}
