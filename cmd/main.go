package main

import (
	"bitbucket.org/novatechnologies/common/events/topics"
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"bitbucket.org/novatechnologies/ohlcv/api/http"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/deal"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"bitbucket.org/novatechnologies/ohlcv/infra/mongo"
	"context"
	"fmt"
	"google.golang.org/protobuf/proto"
	"os"
	"os/signal"
)

func main() {
	ctx := infra.GetContext()
	conf := infra.SetConfig(ctx, "./config/.env")

	consumer := infra.NewConsumer(ctx, conf.KafkaConfig)

	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)
	dealCollection := mongo.GetCollection(ctx, mongoDbClient, conf.MongoDbConfig)
	dealService := deal.NewService(dealCollection, domain.GetAvailableMarkets())
	candleService := candle.NewService(dealCollection, domain.GetAvailableMarkets(), domain.GetAvailableIntervals())

	server := http.NewServer(candleService)
	server.Start(ctx)

	go func() {
		consumer.Consume(ctx, fmt.Sprintf("%s%s", conf.KafkaConfig.TopicPrefix, topics.MatcherMDDeals), func(ctx context.Context, msg []byte) error {

			dealMessage := matcher.Deal{}
			if er := proto.Unmarshal(msg, &dealMessage); er != nil {
				logger.FromContext(ctx).WithField("method", "consumer.deals.Unmarshal").Errorf("%v", er)
				os.Exit(1)
			}

			d, _ := dealService.SaveDeal(ctx, dealMessage)
			candleService.PushUpdatedCandleEvent(ctx, d.Market)

			return nil
		})
	}()

	candleService.CronCandleGenerationStart(ctx)

	//shutdown
	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, os.Interrupt)

	_ = <-signalCh

	server.Stop(ctx)

	return
}
