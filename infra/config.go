package infra

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"context"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type KafkaConfig struct {
	Host          string `envconfig:"KAFKA_HOST" required:"true"`
	ConsumerCount int    `envconfig:"KAFKA_CONSUMER_COUNT" required:"true"`
	TopicPrefix   string `envconfig:"KAFKA_TOPIC_PREFIX" required:"true" default:"master"`
	SslFlag       bool   `envconfig:"KAFKA_SSL_FLAG" required:"true" default:"false"`
}

type MongoDbConfig struct {
	Host               string `envconfig:"MONGODB_HOST" required:"true"`
	DbName             string `envconfig:"MONGODB_NAME" required:"true"`
	DealCollectionName string `envconfig:"MONGODB_DEAL_COLLECTION_NAME" required:"true"`
	TimeOut            int    `envconfig:"MONGODB_TIMEOUT" required:"true"`
	User               string `envconfig:"MONGODB_USER" required:"true"`
	Password           string `envconfig:"MONGODB_PASSWORD" required:"true"`
}

type CentrifugeConfig struct {
	Host  string `envconfig:"CENTRIFUGE_HOST" required:"true"`
	Token string `envconfig:"CENTRIFUGE_TOKEN" required:"true"`
}

// CryptoKeyInPEM is string alias just explicitly informing of PEM format:
// usage https://tools.ietf.org/html/rfc7468
type CryptoKeyInPEM = string

type CentrifugoClientConfig struct {
	Debug bool   `envconfig:"DEBUG" default:"false"`
	Addr  string `envconfig:"CENTRIFUGE_HOST" required:"true"`

	// ServerAPIKey mostly uses for publishing/broadcasting data.
	// See: https://centrifugal.dev/docs/server/server_api
	ServerAPIKey    string `envconfig:"CENTRIFUGE_TOKEN" required:"true"`
	ServerAPIPrefix string `envconfig:"CENTRIFUGO_API_PREFIX" required:"/api"`

	// SignTokenKey mostly uses for subscribing on private channels.
	// See: https://centrifugal.dev/docs/server/private_channels
	SignTokenKey CryptoKeyInPEM `envconfig:"CENTRIFUGO_SIGN_TOKEN_KEY"`
	WSPrefix     string         `envconfig:"CENTRIFUGO_WS_PREFIX" default:"/connection/websocket"`
	//VerifyTokenKey CryptoKeyInPEM `envconfig:"CENTRIFUGO_VERIFY_TOKEN_KEY"`

}

type Config struct {
	KafkaConfig            KafkaConfig
	MongoDbConfig          MongoDbConfig
	CentrifugeConfig       CentrifugeConfig
	CentrifugoClientConfig CentrifugoClientConfig
}

func SetConfig(configPath string) Config {
	err := godotenv.Load(configPath)
	if err != nil {
		panic(err)
	}

	return Parse()
}

func Parse() Config {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		logger.DefaultLogger.
			WithField("err", err).
			Errorf("failed to load configuration")
		panic(err)
	}

	return cfg
}

func GetContext() context.Context {
	return context.Background()
}
