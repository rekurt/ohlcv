package tests

import (
	"log"
	"os"
	"os/signal"
	"testing"
	"time"

	mongo2 "go.mongodb.org/mongo-driver/mongo"

	"bitbucket.org/novatechnologies/ohlcv/infra/centrifuge"

	"bitbucket.org/novatechnologies/interfaces/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bitbucket.org/novatechnologies/ohlcv/api/http"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/deal"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"bitbucket.org/novatechnologies/ohlcv/infra/broker"
	"bitbucket.org/novatechnologies/ohlcv/infra/mongo"
)

func TestForNewCollection(t *testing.T) {
	ctx := infra.GetContext()
	conf := infra.SetConfig("../config/.env")

	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)
	mongo.InitMinutesCollection(ctx, mongoDbClient, conf.MongoDbConfig)

	/*minuteCandleCollection*/ _ = mongo.GetCollection(
		ctx,
		mongoDbClient,
		conf.MongoDbConfig,
		conf.MongoDbConfig.MinuteCandleCollectionName,
	)

	/*testCandle1*/ _ = &domain.Candle{
		Open:      domain.MustParseDecimal("500"),
		High:      domain.MustParseDecimal("500"),
		Low:       domain.MustParseDecimal("500"),
		Close:     domain.MustParseDecimal("500"),
		Volume:    domain.MustParseDecimal("500"),
		Timestamp: time.Now().Truncate(time.Minute),
	}

}

func TestSaveDeal(t *testing.T) {
	ctx := infra.GetContext()
	conf := infra.SetConfig("../config/.env")
	eventsBroker := broker.NewInMemory()

	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)
	// mongo.InitDealsCollection(ctx, mongoDbClient, conf.MongoDbConfig)
	dealCollection := mongo.GetCollection(
		ctx,
		mongoDbClient,
		conf.MongoDbConfig,
		conf.MongoDbConfig.DealCollectionName,
	)

	dealService := deal.NewService(
		dealCollection,
		getTestMarkets(),
		broker.NewInMemory(),
	)
	market := "BTC-USDT"

	d1 := &matcher.Deal{
		Id:           "1234567",
		Market:       market,
		MakerOrderId: "12345",
		TakerOrderId: "12345",
		CreatedAt:    time.Now().Unix(),
		Price:        "102.300",
		Amount:       "0.0031",
	}
	_, err := dealService.SaveDeal(ctx, d1)

	d2 := &matcher.Deal{
		Id:           "1234567",
		Market:       market,
		MakerOrderId: "12345",
		TakerOrderId: "12345",
		CreatedAt:    time.Now().Add(time.Minute * 5).Unix(),
		Price:        "152.300",
		Amount:       "0.0031",
	}
	_, err = dealService.SaveDeal(ctx, d2)

	if err != nil {
		t.Failed()
	}

	d3 := &matcher.Deal{
		Id:           "1234567",
		Market:       market,
		MakerOrderId: "12345",
		TakerOrderId: "12345",
		CreatedAt:    time.Now().Unix(),
		Price:        "52.300",
		Amount:       "0.0121",
	}
	_, err = dealService.SaveDeal(ctx, d3)

	if err != nil {
		t.Fail()
	}

	candleService := InitCandleService(conf, dealCollection, eventsBroker)
	from := time.Now().Add(-5 * time.Minute)
	to := time.Now()
	chart5Min, _ := candleService.GetChart(
		ctx,
		market,
		domain.Candle5MResolution,
		from,
		to,
	)
	currentChart, _ := candleService.GetCurrentCandle(
		ctx,
		market,
		domain.Candle5MResolution,
	)
	// assert.Equal(t, a, b, "The two words should be the same.")
	log.Print(currentChart, chart5Min)
}

func initCandleService(
	conf infra.Config,
	dealsCollection *mongo2.Collection,
	minuteCandleCollection *mongo2.Collection,
) *candle.Service {
	eventsBroker := broker.NewInMemory()
	broadcaster := centrifuge.NewBroadcaster(
		centrifuge.NewPublisher(conf.CentrifugeConfig),
		eventsBroker,
	)
	broadcaster.SubscribeForCharts()

	return candle.NewService(
		&candle.Storage{DealsDbCollection: dealsCollection, CandleDbCollection: minuteCandleCollection},
		new(candle.Agregator),
		domain.GetAvailableMarkets(),
		domain.GetAvailableResolutions(),
		broker.NewInMemory(),
	)
}

func TestDealGenerator(t *testing.T) {
	ctx := infra.GetContext()
	conf := infra.SetConfig("../config/.env")

	eventsBroker := broker.NewInMemory()
	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)

	dealCollection := mongo.GetCollection(
		ctx,
		mongoDbClient,
		conf.MongoDbConfig,
		conf.MongoDbConfig.DealCollectionName,
	)
	dealService := deal.NewService(
		dealCollection,
		domain.GetAvailableMarkets(),
		eventsBroker,
	)
	candleService := InitCandleService(conf, dealCollection, eventsBroker)

	candleService.CronCandleGenerationStart(ctx)

	server := http.NewServer(candleService, dealService, 8082)
	server.Start(ctx)

	// shutdown
	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, os.Interrupt)

	_ = <-signalCh
}

func getTestMarkets() map[string]string {
	return map[string]string{
		"string_with_something_id": "BTC/USDT",
	}
}

func Test_GetLastTrades(t *testing.T) {
	ctx := infra.GetContext()
	conf := infra.SetConfig("../config/.env")

	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)
	// mongo.InitDealsCollection(ctx, mongoDbClient, conf.MongoDbConfig)
	dealCollection := mongo.GetCollection(
		ctx,
		mongoDbClient,
		conf.MongoDbConfig,
		conf.MongoDbConfig.DealCollectionName,
	)
	dealService := deal.NewService(
		dealCollection,
		getTestMarkets(),
		broker.NewInMemory(),
	)
	trades, err := dealService.GetLastTrades(ctx, "ETH/LTC", 10)
	require.NoError(t, err)
	assert.Len(t, trades, 10)
	for _, tr := range trades {
		assert.Equal(t, "ETH/LTC", tr.Data.Market)
	}
}
