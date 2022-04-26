package infra

import (
	"context"
	"fmt"

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

type Config struct {
	KafkaConfig      KafkaConfig
	MongoDbConfig    MongoDbConfig
	CentrifugeConfig CentrifugeConfig
}

func SetConfig(configPath string) Config {
	err := godotenv.Load(configPath)
	if err != nil {
		panic(err)
	}

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		fmt.Println("msg", "failed to load configuration", "err", err)
		panic(err)
	}
	return cfg
}

func GetContext() context.Context {
	return context.Background()
}
