package main

import (
	"context"
	"fmt"

	"bitbucket.org/novatechnologies/ohlcv/internal/model"
	"bitbucket.org/novatechnologies/ohlcv/internal/repository"
	"bitbucket.org/novatechnologies/ohlcv/internal/service"
	"bitbucket.org/novatechnologies/ohlcv/protocol/ohlcv"

	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"bitbucket.org/novatechnologies/common/events/topics"
	"bitbucket.org/novatechnologies/common/infra/logger"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"bitbucket.org/novatechnologies/ohlcv/client/market"

	"bitbucket.org/novatechnologies/ohlcv/api/http"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"bitbucket.org/novatechnologies/ohlcv/infra/broker"
	"bitbucket.org/novatechnologies/ohlcv/infra/centrifuge"
	"bitbucket.org/novatechnologies/ohlcv/infra/mongo"
	"bitbucket.org/novatechnologies/ohlcv/internal/server"
)

func main() {
	ctx := infra.GetContext()
	conf := infra.SetConfig("./config/.env")

	consumer := infra.NewConsumer(ctx, conf.KafkaConfig)
	eventsBroker := broker.NewInMemory()
	fmt.Println(model.GetAvailableResolutions())
	marketsMap, marketsInfo := buildAvailableMarkets(conf)
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

	dealRepository := repository.NewDeal(dealsCollection, marketsMap, marketsInfo)
	dealService := service.NewDeal(dealRepository, marketsMap)

	go dealService.LoadCache(ctx)

	// Start consuming, preparing, savFApiV3Ticker24hrGeting deals into DB and notifying others.
	dealsTopic := conf.KafkaConfig.TopicPrefix + "_" + topics.MatcherMDDeals

	candleService := candle.NewService(&candle.Storage{DealsDbCollection: dealsCollection}, new(candle.Aggregator), eventsBroker)
	klineRepository := repository.NewKline(dealsCollection)
	klineService := service.NewKline(klineRepository)
	updatesStream := make(chan domain.Candle, 512)
	go listenCurrentCandlesUpdates(ctx, updatesStream, eventsBroker, marketsMap)
	currentCandles := initCurrentCandles(ctx, candleService, marketsMap, updatesStream)
	dealService.RunConsuming(ctx, consumer, dealsTopic, currentCandles)

	httpServer := http.NewServer(candleService, dealService, conf)
	httpServer.Start(ctx)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", conf.GRPCConfig.Port))
	if err != nil {
		log.Fatal(err)
	}
	ohlcvSrv := server.NewOhlcv(
		service.NewCandle(repository.NewCandle(dealsCollection)),
		klineService,
		dealService,
	)

	s := grpc.NewServer()
	ohlcv.RegisterOHLCVServiceServer(s, ohlcvSrv)

	reflection.Register(s)
	go func() {
		err = s.Serve(listener)
		if err != nil {
			log.Fatal(err)
		}
	}()
	//shutdown
	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, os.Interrupt)
	_ = <-signalCh
	httpServer.Stop(ctx)
	s.GracefulStop()
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
	count := 0
	started := time.Now()
	for marketId, marketName := range marketsMap {
		for _, resolution := range model.GetAvailableResolutions() {
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
	logger.FromContext(ctx).
		WithField("count", count).
		WithField("elapsed", time.Since(started).String()).
		Infof("initiated candles from MongoDb")
	return candles
}

func buildAvailableMarkets(conf infra.Config) (map[string]string, []market.Market) {
	marketClient, err := market.New(
		market.Config{ServerURL: conf.ExchangeMarketsServerURL, ServerTLS: conf.ExchangeMarketsServerSSL},
		market.NewErrorProcessor(map[string]string{}),
		map[interface{}]market.Option{},
		conf.ExchangeMarketsToken,
	)
	if err != nil {
		log.Fatal("can't market.NewKline:" + err.Error())
	}
	markets, err := marketClient.List(context.Background())
	if err != nil {
		log.Fatal("can't marketClient.List:" + err.Error())
	}
	return domain.GetAvailableMarketsMap(markets), markets
}
