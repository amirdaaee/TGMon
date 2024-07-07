package cmd

import (
	"github.com/amirdaaee/TGMon/config"
	"go.uber.org/zap"
)

func SetupLogger() {
	var ll_ *zap.Logger
	var err_ error

	if config.Config().DevMode {
		ll_, err_ = zap.NewDevelopment()
	} else {
		ll_, err_ = zap.NewProduction()
	}
	ll := zap.Must(ll_, err_)
	zap.ReplaceGlobals(ll)
}
func Setup() {
	SetupLogger()
}

func panpan(err error) {
	zap.L().Sugar().Panic(err)
}
