package log

import (
	"sync"

	"go.uber.org/zap"
)

type LogModule string

const (
	DBModule     LogModule = "db"
	FacadeModule LogModule = "facade"
	TlgModule    LogModule = "tlg"
	BotModule    LogModule = "bot"
	StreamModule LogModule = "stream"
	WebModule    LogModule = "web"
	FuseModule   LogModule = "fuse"
	CmdModule    LogModule = "cmd"
	WorkerModule LogModule = "worker"
	ConfigModule LogModule = "config"
)

var loggerOnce sync.Once

func GetLogger(module LogModule) *zap.Logger {
	return zap.L().Named(string(module))
}

// Named returns a module logger with an additional component name.
func Named(module LogModule, component string) *zap.Logger {
	return GetLogger(module).Named(component)
}

func Setup(levelStr string, dev bool) {
	loggerOnce.Do(func() {

		var llCfg zap.Config
		if dev {
			llCfg = zap.NewDevelopmentConfig()
		} else {
			llCfg = zap.NewProductionConfig()
		}
		if levelStr != "" {
			level, err := zap.ParseAtomicLevel(levelStr)
			if err != nil {
				zap.L().Warn("can not parse log level; using default", zap.String("level", levelStr), zap.String("default", llCfg.Level.String()), zap.Error(err))
			} else {
				llCfg.Level = level
			}
		}
		logger, _ := llCfg.Build(zap.AddStacktrace(zap.ErrorLevel))
		zap.ReplaceGlobals(logger)
	})
}
