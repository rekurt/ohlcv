package main

import (
	"bitbucket.org/novatechnologies/common/events/topics"
	"fmt"
	"os"
	"os/signal"

	"bitbucket.org/novatechnologies/ohlcv/api/http"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/deal"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"bitbucket.org/novatechnologies/ohlcv/infra/broker"
	"bitbucket.org/novatechnologies/ohlcv/infra/centrifuge"
	"bitbucket.org/novatechnologies/ohlcv/infra/mongo"
)

func main() {
	ctx := infra.GetContext()
	conf := infra.SetConfig("./config/.env")

	consumer := infra.NewConsumer(ctx, conf.KafkaConfig)
	eventsBroker := broker.NewInMemory()
	fmt.Println(domain.GetAvailableResolutions())

	broadcaster := centrifuge.NewBroadcaster(
		centrifuge.NewPublisher(conf.CentrifugeConfig),
		eventsBroker,
	)
	broadcaster.SubscribeForCharts()

	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)

	minuteCandleCollection := mongo.GetOrCreateMinutesCollection(
		ctx,
		mongoDbClient,
		conf.MongoDbConfig,
	)

	dealsCollection := mongo.GetOrCreateDealsCollection(
		ctx,
		mongoDbClient,
		conf.MongoDbConfig,
	)

	dealService := deal.NewService(
		dealsCollection,
		domain.GetAvailableMarkets(),
		eventsBroker,
	)
	// Start consuming, preparing, saving deals into DB and notifying others.
	dealsTopic := conf.KafkaConfig.TopicPrefix + "_" + topics.MatcherMDDeals
	dealService.RunConsuming(ctx, consumer, dealsTopic)

	candleService := candle.NewService(
		&candle.Storage{DealsDbCollection: dealsCollection, CandleDbCollection: minuteCandleCollection},
		new(candle.Agregator),
		domain.GetAvailableMarkets(),
		domain.GetAvailableResolutions(),
		eventsBroker,
	)
	candleService.CronCandleGenerationStart(ctx)
	candleService.SubscribeForDeals()

	server := http.NewServer(candleService, dealService, conf.HttpConfig.Port)
	server.Start(ctx)

	//shutdown
	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, os.Interrupt)
	_ = <-signalCh
	server.Stop(ctx)

	return
}
