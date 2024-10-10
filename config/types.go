package config

type configType struct {
	AppID           int      `env:"APP_ID,required"`
	AppHash         string   `env:"APP_HASH,required"`
	TGSocksProxy    string   `env:"TG_SOCKS_PROXY"`
	BotToken        string   `env:"BOT_TOKEN,required"`
	WorkerTokens    []string `env:"WORKER_TOKENS,notEmpty"`
	SessionDir      string   `env:"SESSION_DIR" envDefault:"sessions"`
	ChannelID       int64    `env:"CHANNEL_ID,required"`
	StreamChunkSize int64    `env:"STREAM_CHUNK_SIZE" envDefault:"1048576"`
	LogLevel        string   `env:"LOG_LEVEL" envDefault:"warning"`
	// ...
	UserName      string `env:"USER_NAME,required"`
	UserPass      string `env:"USER_PASS,required"`
	UserToken     string `env:"USER_TOKEN,required"`
	AccessLogFile string `env:"ACCESS_LOG_FILE"`
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
