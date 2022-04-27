package main

import (
	"os"
	"os/signal"

	"bitbucket.org/novatechnologies/common/events/topics"

	"bitbucket.org/novatechnologies/ohlcv/api/http"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/deal"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"bitbucket.org/novatechnologies/ohlcv/infra/inmemo"
	"bitbucket.org/novatechnologies/ohlcv/infra/mongo"
)

func main() {
	ctx := infra.GetContext()
	conf := infra.SetConfig("./config/.env")

	consumer := infra.NewConsumer(ctx, conf.KafkaConfig)
	marketDataBus := inmemo.NewInMemory()

	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)
	//mongo.InitDealCollection(ctx, mongoDbClient, conf.MongoDbConfig)
	dealCollection := mongo.GetCollection(
		ctx,
		mongoDbClient,
		conf.MongoDbConfig,
	)

	dealService := deal.NewService(
		dealCollection,
		domain.GetAvailableMarkets(),
		marketDataBus,
	)

	candleService := candle.NewService(
		dealCollection,
		domain.GetAvailableMarkets(),
		domain.GetAvailableIntervals(),
		marketDataBus,
	)

	server := http.NewServer(candleService, dealService)
	server.Start(ctx)

	// Start consuming, preparing, saving deals into DB and notifying others.
	dealsTopic := conf.KafkaConfig.TopicPrefix + "_" + topics.MatcherMDDeals
	go dealService.RunConsuming(ctx, consumer, dealsTopic)

	candleService.CronCandleGenerationStart(ctx)

	//shutdown
	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, os.Interrupt)

	_ = <-signalCh

	server.Stop(ctx)

	return
}
