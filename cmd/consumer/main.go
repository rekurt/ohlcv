package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
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
	dealService.RunConsuming(ctx, consumer, dealsTopic)

	candleService := candle.NewService(
		&candle.Storage{DealsDbCollection: dealsCollection},
		new(candle.Aggregator),
		marketsMap,
		domain.GetAvailableResolutions(),
		eventsBroker,
	)
	initCurrentCandles(ctx, candleService, marketsMap)
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
				log.Fatal("can't chartToCurrentCandle to initCurrentCandles:" + err.Error())
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

func chartToCurrentCandle(chart *domain.Chart, resolution string) (candle.CurrentCandle, error) {
	if chart == nil {
		openTime := time.Unix((&candle.Aggregator{}).GetCurrentResolutionStartTimestamp(resolution, time.Now()), 0).UTC()
		return candle.CurrentCandle{
			OpenTime:  openTime,
			CloseTime: openTime.Add(domain.StrResolutionToDuration(resolution)).UTC(),
		}, nil
	}
	if len(chart.O) != 1 {
		return candle.CurrentCandle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.O))
	}
	open, err := strconv.ParseFloat(chart.O[0].String(), 64)
	if err != nil {
		return candle.CurrentCandle{}, err
	}
	if len(chart.H) != 1 {
		return candle.CurrentCandle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.H))
	}
	high, err := strconv.ParseFloat(chart.H[0].String(), 64)
	if err != nil {
		return candle.CurrentCandle{}, err
	}
	if len(chart.L) != 1 {
		return candle.CurrentCandle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.L))
	}
	low, err := strconv.ParseFloat(chart.L[0].String(), 64)
	if err != nil {
		return candle.CurrentCandle{}, err
	}
	if len(chart.C) != 1 {
		return candle.CurrentCandle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.C))
	}
	closePrice, err := strconv.ParseFloat(chart.C[0].String(), 64)
	if err != nil {
		return candle.CurrentCandle{}, err
	}
	if len(chart.V) != 1 {
		return candle.CurrentCandle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.V))
	}
	volume, err := strconv.ParseFloat(chart.V[0].String(), 64)
	if err != nil {
		return candle.CurrentCandle{}, err
	}
	if len(chart.T) != 1 {
		return candle.CurrentCandle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.T))
	}
	openTime := time.Unix(chart.T[0], 0).UTC()
	return candle.CurrentCandle{
		Symbol:    chart.Symbol,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     closePrice,
		Volume:    volume,
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
