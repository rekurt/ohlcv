package main

import (
	"bitbucket.org/novatechnologies/common/events/topics"
	"bitbucket.org/novatechnologies/ohlcv/infra/centrifuge"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"os"
	"os/signal"
	"time"

	"bitbucket.org/novatechnologies/ohlcv/client/market"

	"bitbucket.org/novatechnologies/ohlcv/api/http"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/deal"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"bitbucket.org/novatechnologies/ohlcv/infra/broker"
	"bitbucket.org/novatechnologies/ohlcv/infra/mongo"
)

func main() {
	ctx := infra.GetContext()
	conf := infra.SetConfig("./config/.env")

	consumer := infra.NewConsumer(ctx, conf.KafkaConfig)
	eventsBroker := broker.NewInMemory()
	fmt.Println(domain.GetAvailableResolutions())
	marketsMap := buildAvailableMarkets(conf)
	broadcaster := centrifuge.NewBroadcaster(
		centrifuge.NewPublisher(conf.CentrifugeConfig),
		eventsBroker,
		marketsMap,
	)
	broadcaster.SubscribeForCharts()

	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)

	dealsCollection := mongo.GetOrCreateDealsCollection(
		ctx,
		mongoDbClient,
		conf.MongoDbConfig,
	)

	dealService := deal.NewService(
		dealsCollection,
		marketsMap,
		eventsBroker,
	)
	// Start consuming, preparing, saving deals into DB and notifying others.
	dealsTopic := conf.KafkaConfig.TopicPrefix + "_" + topics.MatcherMDDeals

	candleService := candle.NewService(
		&candle.Storage{DealsDbCollection: dealsCollection},
		new(candle.Aggregator),
		marketsMap,
		domain.GetAvailableResolutions(),
		eventsBroker,
	)
	currentCandles := initCurrentCandles(ctx, candleService, marketsMap)
	go listenCurrentCandlesUpdates(ctx, currentCandles.GetUpdates(), eventsBroker)
	dealService.RunConsuming(ctx, consumer, dealsTopic, currentCandles)
	candleService.CronCandleGenerationStart(ctx)
	candleService.SubscribeForDeals()

	server := http.NewServer(candleService, dealService, conf)
	server.Start(ctx)

	//shutdown
	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, os.Interrupt)
	_ = <-signalCh
	server.Stop(ctx)

	return
}

func listenCurrentCandlesUpdates(ctx context.Context, updates <-chan domain.Candle, eventsBroker *broker.EventsInMemory) {
	for upd := range updates {
		charts := []*domain.Chart{
			{
				Symbol: upd.Symbol,
				O:      []primitive.Decimal128{upd.Open},
				H:      []primitive.Decimal128{upd.High},
				L:      []primitive.Decimal128{upd.Low},
				C:      []primitive.Decimal128{upd.Close},
				V:      []primitive.Decimal128{upd.Volume},
				T:      []int64{upd.OpenTime.Unix()},
			},
		}
		eventsBroker.Publish(domain.EvTypeCharts, domain.NewEvent(ctx, charts))
	}
}

func initCurrentCandles(ctx context.Context, service *candle.Service, marketsMap map[string]string) candle.CurrentCandles {
	candles := candle.NewCurrentCandles(ctx)
	count := 0
	started := time.Now()
	for marketId, marketName := range marketsMap {
		for _, resolution := range domain.GetAvailableResolutions() {
			chart, err := service.GetCurrentCandle(ctx, marketName, resolution)
			if err != nil {
				log.Fatal("can't GetCurrentCandle to initCurrentCandles:" + err.Error())
			}
			currentCandle, err := domain.ChartToCurrentCandle(chart, resolution)
			if err != nil {
				log.Fatalf("can't chartToCurrentCandle to initCurrentCandles: %s, chart: %+v", err, chart)
			}
			err = candles.AddCandle(marketId, resolution, currentCandle)
			if err != nil {
				log.Fatal("can't AddCandle to initCurrentCandles:" + err.Error())
			}
			count++
		}
	}
	fmt.Printf("initiated %d candles from MongoDb for %s", count, time.Since(started))
	return candles
}

func buildAvailableMarkets(conf infra.Config) map[string]string {
	marketClient, err := market.New(
		market.Config{ServerURL: conf.ExchangeMarketsServerURL, ServerTLS: conf.ExchangeMarketsServerSSL},
		market.NewErrorProcessor(map[string]string{}),
		map[interface{}]market.Option{},
		conf.ExchangeMarketsToken,
	)
	if err != nil {
		log.Fatal("can't market.New:" + err.Error())
	}
	markets, err := marketClient.List(context.Background())
	if err != nil {
		log.Fatal("can't marketClient.List:" + err.Error())
	}
	return domain.GetAvailableMarketsMap(markets)
}
