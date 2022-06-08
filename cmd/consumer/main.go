package main

import (
	"bitbucket.org/novatechnologies/common/events/topics"
	"bitbucket.org/novatechnologies/ohlcv/infra/centrifuge"
	"context"
	"fmt"
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
	go listenCurrentCandlesUpdates(currentCandles.GetUpdates())
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

func listenCurrentCandlesUpdates(updates <-chan domain.Candle) {
	for update := range updates {
		fmt.Printf("CurrentCandle: %+v\n", update)
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
			currentCandle, err := chartToCurrentCandle(chart, resolution)
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

func chartToCurrentCandle(chart *domain.Chart, resolution string) (domain.Candle, error) {
	if chart == nil {
		openTime := time.Unix((&candle.Aggregator{}).GetResolutionStartTimestampByTime(resolution, time.Now()), 0).UTC()
		return domain.Candle{
			OpenTime:  openTime,
			CloseTime: openTime.Add(domain.StrResolutionToDuration(resolution)).UTC(),
		}, nil
	}
	if len(chart.O) == 0 {
		return domain.Candle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.O))
	}
	if len(chart.H) == 0 {
		return domain.Candle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.H))
	}
	if len(chart.L) == 0 {
		return domain.Candle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.L))
	}
	if len(chart.C) == 0 {
		return domain.Candle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.C))
	}
	if len(chart.V) == 0 {
		return domain.Candle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.V))
	}
	if len(chart.T) == 0 {
		return domain.Candle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.T))
	}
	openTime := time.Unix(chart.T[len(chart.T)-1], 0).UTC()
	return domain.Candle{
		Symbol:    chart.Symbol,
		Open:      chart.O[len(chart.O)-1],
		High:      chart.H[len(chart.H)-1],
		Low:       chart.L[len(chart.L)-1],
		Close:     chart.C[len(chart.C)-1],
		Volume:    chart.V[len(chart.V)-1],
		OpenTime:  openTime,
		CloseTime: openTime.Add(domain.StrResolutionToDuration(resolution)).UTC(),
	}, nil
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
