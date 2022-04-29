package infra

import (
	"context"

	pubsub "bitbucket.org/novatechnologies/common/events"
	"bitbucket.org/novatechnologies/common/events/kafka"
	"bitbucket.org/novatechnologies/common/infra/logger"
	"golang.org/x/sync/errgroup"
)

func NewPublisher(ctx context.Context, cfg KafkaConfig) (
	pubsub.Publisher, error,
) {
	log := logger.FromContext(ctx).
		WithField("component", "publisher").
		WithField("broker", "kafka")

	kPub, err := kafka.NewPublisher(log, []string{cfg.Host}, cfg.SslFlag)
	if err != nil {
		log.Errorf("[kafka]NewPublisher failed with err: %v", err)
		return nil, err
	}
	pub, err := pubsub.NewWrappedPublihser(kPub)
	if err != nil {
		log.Errorf("[kafka]NewPublisher failed with err: %v", err)
		return nil, err
	}

	return pub, nil
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
	consumer, err := pubsub.NewWrappedSubscriber(
		kSub, group, pubsub.WSubscriberConfig{
			Name:         "OhlcvConsumer",
			WorkersCount: config.ConsumerCount,
		},
	)

	return consumer
}
