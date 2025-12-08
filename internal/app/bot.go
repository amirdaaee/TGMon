//go:build wireinject
// +build wireinject

package app

import (
	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/config"
	"github.com/google/wire"
)

func InitializeBot(cfg *config.ConfigType) (*bot.Bot, error) {
	wire.Build(NewBot, NewDbContainer, TgProviderSet, FacadeProviderSet)
	return nil, nil
}
