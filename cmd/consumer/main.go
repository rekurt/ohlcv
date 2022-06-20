package main

import (
	"bitbucket.org/novatechnologies/common/events/topics"
	"bitbucket.org/novatechnologies/common/infra/logger"
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
	updatesStream := make(chan domain.Candle, 512)
	go listenCurrentCandlesUpdates(ctx, updatesStream, eventsBroker, marketsMap)
	currentCandles := initCurrentCandles(ctx, candleService, marketsMap, updatesStream)
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

func listenCurrentCandlesUpdates(ctx context.Context, updates <-chan domain.Candle, eventsBroker *broker.EventsInMemory, marketsMap map[string]string) {
	chartStream := make(chan *domain.Chart)
	defer close(chartStream)
	batchStream := domain.Microbatching(ctx, chartStream, 10)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case batch := <-batchStream:
				eventsBroker.Publish(domain.EvTypeCharts, domain.NewEvent(ctx, batch))
			}
		}
	}()
	for upd := range updates {
		symbol := marketsMap[upd.Symbol]
		if symbol == "" {
			symbol = upd.Symbol
		}
		chart := domain.Chart{
			Symbol:     symbol,
			Resolution: upd.Resolution,
			O:          []primitive.Decimal128{upd.Open},
			H:          []primitive.Decimal128{upd.High},
			L:          []primitive.Decimal128{upd.Low},
			C:          []primitive.Decimal128{upd.Close},
			V:          []primitive.Decimal128{upd.Volume},
			T:          []int64{upd.OpenTime.Unix()},
		}
		select {
		case <-ctx.Done():
			return
		case chartStream <- &chart:
		}
	}
}

func initCurrentCandles(ctx context.Context, service *candle.Service, marketsMap map[string]string, updatesStream chan domain.Candle) candle.CurrentCandles {
	candles := candle.NewCurrentCandles(ctx, updatesStream)
	var keys []string
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
			keys = append(keys, marketId+"-"+resolution)
			count++
		}
	}
	logger.FromContext(ctx).
		WithField("count", count).
		WithField("elapsed", time.Since(started).String()).
		WithField("keys", keys).
		Infof("initiated candles from MongoDb")
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
