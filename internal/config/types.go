package config

type HttpConfigType struct {
	UserName     string   `env:"USER_NAME,required"`
	UserPass     string   `env:"USER_PASS,required"`
	ApiToken     string   `env:"API_TOKEN,required"`
	Swagger      bool     `env:"SWAGGER" envDefault:"false"`
	CoresAllowed []string `env:"CORES_ALLOWED_ORIGINS"`
	ListenAddr   string   `env:"LISTEN_ADDR" envDefault:":8080"`
}
type ConfigType struct {
	AppID           int            `env:"APP_ID,required"`
	AppHash         string         `env:"APP_HASH,required"`
	TGSocksProxy    string         `env:"TG_SOCKS_PROXY"`
	BotToken        string         `env:"BOT_TOKEN,required"`
	WorkerTokens    []string       `env:"WORKER_TOKENS,notEmpty"`
	WorkerCacheRoot string         `env:"WORKER_CACHE_ROOT,required"`
	SessionDir      string         `env:"SESSION_DIR" envDefault:"sessions"`
	ChannelID       int64          `env:"CHANNEL_ID,required"`
	LogLevel        string         `env:"LOG_LEVEL" envDefault:"warning"`
	HttpConfig      HttpConfigType `envPrefix:"HTTP_CONFIG__"`
	// ...
	KeepDupFiles bool `env:"KEEP_DUP_FILE"`
	// ...
	MinioEndpoint  string `env:"MINIO_ENDPOINT,required"`
	MinioAccessKey string `env:"MINIO_ACCESS_KEY,required"`
	MinioSecretKey string `env:"MINIO_SECRET_KEY,required"`
	MinioBucket    string `env:"MINIO_BUCKET,required"`
	MinioSecure    bool   `env:"MINIO_SECURE" envDefault:"true"`
	// ...
	MongoDBUri  string `env:"MONGODB_URI,required"`
	MongoDBName string `env:"MONGODB_DB_NAME,required"`
}
