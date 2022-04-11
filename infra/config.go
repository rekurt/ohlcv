package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

var config Config

type KafkaConfig struct {
	Host          string `envconfig:"KAFKA_HOST" required:"true"`
	ConsumerCount int    `envconfig:"KAFKA_CONSUMER_COUNT" required:"true"`
	TopicPrefix   string `envconfig:"KAFKA_TOPIC_PREFIX" required:"true" default:"master_"`
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
	Host string `envconfig:"CENTRIFUGE_HOST" required:"true"`
}

type Config struct {
	KafkaConfig      KafkaConfig
	MongoDbConfig    MongoDbConfig
	CentrifugeConfig CentrifugeConfig
}

func SetConfig(ctx context.Context, configPath string) Config {
	err := godotenv.Load(configPath)
	if err != nil {
		panic(err)
	}

	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		fmt.Println("msg", "failed to load configuration", "err", err)
		panic(err)
	}
	bs, _ := json.Marshal(cfg)
	fmt.Println("CONFIG:", string(bs))

	return cfg
}

func GetContext() context.Context {
	return context.Background()
}
