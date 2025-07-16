package config

import (
	"os"
	"sync"

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
		if _, error := os.Stat(".env"); !os.IsNotExist(error) {
			zap.S().Info("found .env file")
			if err := godotenv.Load(); err != nil {
				zap.S().Panic(err)
			}
		} else {
			zap.S().Info("no .env file found")
		}
		configInstance = &ConfigType{}
		if err := env.Parse(configInstance); err != nil {
			panic(err)
		}
	}
	return configInstance
}
