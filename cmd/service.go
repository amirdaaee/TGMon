package cmd

import (
	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/web"
	"go.uber.org/zap"
)

func RunService() {
	ll := zap.L()
	_, err := bot.StartWorkers(ll)
	if err != nil {
		ll.Sugar().Panic(err)
	}
	mainBot, err := bot.StartMainBot(ll)
	if err != nil {
		ll.Sugar().Panic(err)
	}
	go web.Start()
	mainBot.Idle()
}
