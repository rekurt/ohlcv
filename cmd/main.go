package main

import (
	"bitbucket.org/novatechnologies/common/events/topics"
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"bitbucket.org/novatechnologies/ohlcv/api/http"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/deal"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"bitbucket.org/novatechnologies/ohlcv/infra/mongo"
	"context"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
	"os"
	"os/signal"
	"time"
)

func main() {
	ctx := infra.GetContext()
	ctx, _ = context.WithTimeout(ctx, time.Second*15)
	conf := infra.SetConfig(ctx, "./config/.env")

	group, ctx := errgroup.WithContext(ctx)

	consumer := infra.NewConsumer(ctx, conf.KafkaConfig)

	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)
	dealCollection := mongo.GetCollection(ctx, mongoDbClient, conf.MongoDbConfig)
	deal2Collection := mongo.GetCollection(ctx, mongoDbClient, conf.MongoDbConfig)
	dealService := deal.NewService(dealCollection)
	candleService := candle.NewService(deal2Collection, getMarkets())

	server := http.NewServer(candleService)

	group.Go(func() error {
		return consumer.Consume(ctx, topics.MatcherMDDeals, func(ctx context.Context, msg []byte) error {
			dealMessage := matcher.Deal{}
			if er := proto.Unmarshal(msg, &dealMessage); er != nil {
				logger.FromContext(ctx).WithField("method", "consumer.deals.Unmarshal").Errorf("%v", er)
				os.Exit(1)
			}

			dealService.SaveDeal(ctx, dealMessage)
			candleService.PushLastUpdatedCandle(ctx, dealMessage.Market)
			return nil
		})
	})

	candleService.CronCandleGenerationStart(ctx)
	server.Start(ctx)

	//shutdown
	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, os.Interrupt)

	_ = <-signalCh

	server.Stop(ctx)

	return
}

func getMarkets() []string {
	return []string{
/*		"USDT_TRX",
		"USDT_ETH",
		"USDT_LTC",
		"USDT_BCH",
		"BTC_XRP",
		"USDT_BTC",
		"BTC_BCH",
		"ETH_TRX",
		"BTC_ETH",
		"BTC_LINK",
		"USDT_LINK",
		"ETH_LINK",
		"ETH_LTC",
		"BTC_LTC",
		"ETH_XRP",
		"BTC_TRX",*/
		"BTC-USDT",
	}
}
