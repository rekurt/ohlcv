package infra

import (
	"context"

	"bitbucket.org/novatechnologies/common/infra/logger"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type GRPCConfig struct {
	Port int `envconfig:"GRPC_PORT" default:"8183"`
}

type HttpConfig struct {
	Port int `envconfig:"API_PORT" required:"false" default:"8082"`
}

type KafkaConfig struct {
	Host          string `envconfig:"KAFKA_HOST" required:"true"`
	ConsumerCount int    `envconfig:"KAFKA_CONSUMER_COUNT" required:"true"`
	TopicPrefix   string `envconfig:"KAFKA_TOPIC_PREFIX" required:"true" default:"master"`
	SslFlag       bool   `envconfig:"KAFKA_SSL" required:"true" default:"false"`
}

type MongoDbConfig struct {
	ConnectionUrl              string `envconfig:"MONGODB_URL" required:"true"`
	DatabaseName               string `envconfig:"MONGODB_NAME" default:"dbName"`
	TimeOut                    int    `envconfig:"MONGODB_TIMEOUT" required:"true"`
	MinuteCandleCollectionName string `envconfig:"MONGODB_MINUTE_CANDLE_COLLECTION_NAME" required:"true" default:"minutes"`
	DealCollectionName         string `envconfig:"MONGODB_DEAL_COLLECTION_NAME" required:"true"`
}

// CryptoKeyInPEM is string alias just explicitly informing of PEM format:
// usage https://tools.ietf.org/html/rfc7468
type CryptoKeyInPEM = string

type CentrifugeConfig struct {
	Debug bool   `envconfig:"DEBUG" default:"false"`
	Host  string `envconfig:"CENTRIFUGE_HOST" required:"true"`

	// ServerAPIKey mostly uses for publishing/broadcasting data.
	// See: https://centrifugal.dev/docs/server/server_api
	ServerAPIKey    string `envconfig:"CENTRIFUGE_TOKEN" required:"true"`
	ServerAPIPrefix string `envconfig:"CENTRIFUGO_API_PREFIX" required:"/api"`

	// SignTokenKey mostly uses for subscribing on private channels.
	// See: https://centrifugal.dev/docs/server/private_channels
	SignTokenKey CryptoKeyInPEM `envconfig:"CENTRIFUGO_SIGN_TOKEN_KEY"`
	WSPrefix     string         `envconfig:"CENTRIFUGO_WS_PREFIX" default:"/connection/websocket"`
	// VerifyTokenKey CryptoKeyInPEM `envconfig:"CENTRIFUGO_VERIFY_TOKEN_KEY"`
}

type Config struct {
	KafkaConfig              KafkaConfig
	GRPCConfig               GRPCConfig
	MongoDbConfig            MongoDbConfig
	CentrifugeConfig         CentrifugeConfig
	HttpConfig               HttpConfig
	ExchangeMarketsServerURL string `envconfig:"EXCHANGE_MARKETS_SERVER_URL"`
	ExchangeMarketsServerSSL bool   `envconfig:"EXCHANGE_MARKETS_SERVER_SSL" default:"true"`
	ExchangeMarketsToken     string `envconfig:"EXCHANGE_MARKETS_TOKEN"`
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
