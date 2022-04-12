package infra

import (
	pubsub "bitbucket.org/novatechnologies/common/events"
	"bitbucket.org/novatechnologies/common/events/kafka"
	"bitbucket.org/novatechnologies/common/infra/logger"
	"context"
	"golang.org/x/sync/errgroup"
)

func NewProducer(ctx context.Context, host string) interface{} {
	brokers := []string{host}
	log := logger.FromContext(ctx).WithField("m", "main")
	kPub, err := kafka.NewPublisher(log, brokers, false)
	if err != nil {
		log.Errorf("[kafka]NewPublisher failed with err: %v", err)
		panic(err)
	}
	pub, err := pubsub.NewWrappedPublihser(kPub)
	if err != nil {
		log.Errorf("[kafka]NewPublisher failed with err: %v", err)
		panic(err)
	}

	return pub
}

func NewConsumer(ctx context.Context, config KafkaConfig) pubsub.Subscriber {
	group, _ := errgroup.WithContext(ctx)
	brokers := []string{config.Host}
	log := logger.FromContext(ctx).WithField("m", "main")
	kSub, err := kafka.NewSubscriber(log, brokers, config.SslFlag)

	if err != nil {
		log.Errorf("[kafka]NewConsumer failed with err: %v", err)
		panic(err)
	}
	consumer, err := pubsub.NewWrappedSubscriber(kSub, group, pubsub.WSubscriberConfig{
		Name:         "OhlcvConsumer",
		WorkersCount: config.ConsumerCount,
	})

	return consumer
}
