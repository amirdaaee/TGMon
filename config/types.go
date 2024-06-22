package config

import (
	"net/url"
)

type ProxyUri url.URL
type configType struct {
	AppID        int      `env:"APP_ID,required"`
	AppHash      string   `env:"APP_HASH,required"`
	TGSocksProxy ProxyUri `env:"TG_SOCKS_PROXY"`
	BotToken     string   `env:"BOT_TOKEN,required"`
	WorkerTokens []string `env:"WORKER_TOKENS,notEmpty"`
	SessionDir   string   `env:"SESSION_FILE" envDefault:"sessions"`
	ChannelID    int64    `env:"CHANNEL_ID,required"`
	MongoDBUri   string   `env:"MONGODB_URI,required"`
	MongoDBName  string   `env:"MONGODB_DB_NAME,required"`
	ListenURL    string   `env:"LISTEN_URL" envDefault:"0.0.0.0:8081"`
	DevMode      bool     `env:"DEV_MODE" envDefault:"false"`
	AccessLog    string   `env:"ACCESS_LOG" envDefault:"storage/gin.log"`
	UserName     string   `env:"USER_NAME,required"`
	UserPass     string   `env:"USER_PASS,required"`
	UserToken    string   `env:"USER_TOKEN,required"`
}

func (t *ProxyUri) UnmarshalText(text []byte) error {
	u, err := url.Parse(string(text))
	if err != nil {
		return err
	}
	*t = ProxyUri(*u)
	return nil
}
