package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"bitbucket.org/novatechnologies/common/events/topics"
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"google.golang.org/protobuf/proto"

	"bitbucket.org/novatechnologies/ohlcv/api/http"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/deal"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"bitbucket.org/novatechnologies/ohlcv/infra/mongo"
)

func main() {
	ctx := infra.GetContext()
	conf := infra.SetConfig("./config/.env")

	consumer := infra.NewConsumer(ctx, conf.KafkaConfig)

	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)
	dealCollection := mongo.GetCollection(
		ctx,
		mongoDbClient,
		conf.MongoDbConfig,
	)
	dealService := deal.NewService(dealCollection, domain.GetAvailableMarkets())
	candleService := candle.NewService(
		dealCollection,
		domain.GetAvailableMarkets(),
		domain.GetAvailableIntervals(),
	)

	server := http.NewServer(candleService, dealService)
	server.Start(ctx)

	go func() {
		err := func() error {
			topicName := fmt.Sprintf(
				"%s%s%s",
				conf.KafkaConfig.TopicPrefix,
				"_",
				topics.MatcherMDDeals,
			)
			return consumer.Consume(
				ctx,
				topicName,
				func(
					ctx context.Context,
					metadata map[string]string,
					msg []byte,
				) error {

					dealMessage := matcher.Deal{}
					if er := proto.Unmarshal(msg, &dealMessage); er != nil {

						logger.FromContext(ctx).WithField(
							"method",
							"consumer.deals.Unmarshal",
						).Errorf("%v", er)
						os.Exit(1)
					}

					d, _ := dealService.SaveDeal(ctx, dealMessage)
					candleService.PushUpdatedCandleEvent(ctx, d.Market)

					return nil
				},
			)
		}()
		if err != nil {
			logger.FromContext(ctx).WithField(
				"error",
				err.Error(),
			).Errorf("[DealService]Failed Kafka consumer")
		}
	}()

	candleService.CronCandleGenerationStart(ctx)

	//shutdown
	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, os.Interrupt)

	_ = <-signalCh

	server.Stop(ctx)

	return
}
