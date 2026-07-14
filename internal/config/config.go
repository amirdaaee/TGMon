package config

import (
	"os"
	"sync"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

var lock = &sync.Mutex{}
var configInstance *ConfigType

// Config returns a singleton instance of ConfigType, loading environment variables from a .env file if present.
// It uses sync.Mutex to ensure thread-safe initialization and parses environment variables into the ConfigType struct.
func Config() *ConfigType {
	if configInstance == nil {
		lock.Lock()
		defer lock.Unlock()
		ll := log.GetLogger(log.ConfigModule)
		if _, error := os.Stat(".env"); !os.IsNotExist(error) {
			ll.Info("found .env file")
			if err := godotenv.Load(); err != nil {
				ll.With(zap.Error(err)).Fatal("can not load .env file")
			}
		} else {
			ll.Info("no .env file found")
		}
		configInstance = &ConfigType{}
		if err := env.Parse(configInstance); err != nil {
			panic(err)
		}
		ll.Sugar().Infof("config loaded: %+v", configInstance)
	}
	return configInstance
}
