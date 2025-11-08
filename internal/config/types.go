package config

type HttpConfigType struct {
	UserName     string   `env:"USER_NAME,required"`
	UserPass     string   `env:"USER_PASS,required"`
	ApiToken     string   `env:"API_TOKEN,required"`
	Swagger      bool     `env:"SWAGGER" envDefault:"false"`
	CoresAllowed []string `env:"CORES_ALLOWED_ORIGINS"`
	ListenAddr   string   `env:"LISTEN_ADDR" envDefault:":8080"`
}
type TelegramConfigType struct {
	AppID           int      `env:"APP_ID,required"`
	AppHash         string   `env:"APP_HASH,required"`
	TGSocksProxy    string   `env:"TG_SOCKS_PROXY"`
	BotToken        string   `env:"BOT_TOKEN,required"`
	WorkerTokens    []string `env:"WORKER_TOKENS,notEmpty"`
	WorkerCacheRoot string   `env:"WORKER_CACHE_ROOT,required"`
	SessionDir      string   `env:"SESSION_DIR" envDefault:"sessions"`
	ChannelID       int64    `env:"CHANNEL_ID,required"`
}
type MinioConfigType struct {
	Endpoint  string `env:"ENDPOINT,required"`
	AccessKey string `env:"ACCESS_KEY,required"`
	SecretKey string `env:"SECRET_KEY,required"`
	Bucket    string `env:"BUCKET,required"`
	Secure    bool   `env:"SECURE" envDefault:"true"`
}
type MongoDBConfigType struct {
	Uri    string `env:"URI,required"`
	DBName string `env:"DB_NAME,required"`
}
type FuseConfigType struct {
	AllowOther bool `env:"ALLOW_OTHER" envDefault:"true"`
	Debug      bool `env:"DEBUG" envDefault:"false"`
}
type RuntimeConfigType struct {
	LogLevel       string `env:"LOG_LEVEL" envDefault:"warning"`
	KeepDupFiles   bool   `env:"KEEP_DUP_FILE"`
	StreamBuffSize int    `env:"STREAM_BUFF_SIZE" envDefault:"8388608"`
}
type ConfigType struct {
	TelegramConfig TelegramConfigType `envPrefix:"TELEGRAM__"`
	HttpConfig     HttpConfigType     `envPrefix:"HTTP__"`
	MinioConfig    MinioConfigType    `envPrefix:"MINIO__"`
	MongoDBConfig  MongoDBConfigType  `envPrefix:"MONGODB__"`
	FuseConfig     FuseConfigType     `envPrefix:"FUSE__"`
	RuntimeConfig  RuntimeConfigType  `envPrefix:"RUNTIME__"`
}
