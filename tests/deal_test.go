package tests

import (
	"bitbucket.org/novatechnologies/ohlcv/client/market"
	"bitbucket.org/novatechnologies/ohlcv/internal/model"
	"bitbucket.org/novatechnologies/ohlcv/internal/repository"
	"bitbucket.org/novatechnologies/ohlcv/internal/service"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"testing"
	"time"

	"bitbucket.org/novatechnologies/interfaces/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bitbucket.org/novatechnologies/ohlcv/api/http"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"bitbucket.org/novatechnologies/ohlcv/infra/broker"
	"bitbucket.org/novatechnologies/ohlcv/infra/mongo"
)

func TestForNewCollection_manual(t *testing.T) {
	t.Skip()
	ctx := infra.GetContext()
	conf := infra.SetConfig("../config/.env")

	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)
	mongo.InitMinutesCollection(ctx, mongoDbClient, conf.MongoDbConfig)

	/*minuteCandleCollection*/
	_ = mongo.GetCollection(
		ctx,
		mongoDbClient,
		conf.MongoDbConfig,
		conf.MongoDbConfig.MinuteCandleCollectionName,
	)

	/*testCandle1*/
	_ = &domain.Candle{
		Open:     model.MustParseDecimal("500"),
		High:     model.MustParseDecimal("500"),
		Low:      model.MustParseDecimal("500"),
		Close:    model.MustParseDecimal("500"),
		Volume:   model.MustParseDecimal("500"),
		OpenTime: time.Now().Truncate(time.Minute),
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

	dealService := service.NewDeal(repository.NewDeal(dealCollection, getTestMarkets(), nil), getTestMarkets(), make(chan *model.Deal))
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
	chart5Min := candleService.GetChart(
		ctx,
		market,
		model.Candle5MResolution,
		from,
		to,
	)
	currentChart, _ := candleService.GetCurrentCandle(
		ctx,
		market,
		model.Candle5MResolution,
	)
	// assert.Equal(t, a, b, "The two words should be the same.")
	log.Print(currentChart, chart5Min)
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
	dealService := service.NewDeal(repository.NewDeal(dealCollection, GetAvailableMarkets(), nil), GetAvailableMarkets(), make(chan *model.Deal))
	candleService := InitCandleService(conf, dealCollection, eventsBroker)

	server := http.NewServer(candleService, dealService, conf)
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

func Test_GetTickerPriceChangeStatistics(t *testing.T) {
	t.Skip()
	ctx := infra.GetContext()
	conf := infra.SetConfig("../config/.env")

	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)
	dealCollection := mongo.GetCollection(
		ctx,
		mongoDbClient,
		conf.MongoDbConfig,
		conf.MongoDbConfig.DealCollectionName,
	)
	service := service.NewDeal(repository.NewDeal(dealCollection, getTestMarkets(), nil), getTestMarkets(), make(chan *model.Deal))
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*15)
	defer cancelFunc()
	statistics, err := service.GetTickerPriceChangeStatistics(ctx, "")
	require.NoError(t, err)
	for _, s := range statistics {
		// fmt.Printf("%+v\n", s)
		fmt.Println(s.OpenPrice, s.LastPrice, s.PriceChange, s.PriceChangePercent)
	}
}

func Test_GetLastTrades(t *testing.T) {
	t.Skip()
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
	dealService := service.NewDeal(repository.NewDeal(dealCollection, getTestMarkets(), nil), getTestMarkets(), make(chan *model.Deal))
	trades, err := dealService.GetLastTrades(ctx, "ETH/LTC", 10)
	require.NoError(t, err)
	assert.Len(t, trades, 10)
	for _, tr := range trades {
		assert.Equal(t, "ETH/LTC", tr.Data.Market)
	}
}

func buildAvailableMarkets(conf infra.Config) []market.Market {
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
	return markets
}

func Test_GetAvgPrice(t *testing.T) {
	t.Skip()
	ctx := infra.GetContext()
	conf := infra.SetConfig("../config/.env")
	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)
	dealCollection := mongo.GetCollection(
		ctx,
		mongoDbClient,
		conf.MongoDbConfig,
		conf.MongoDbConfig.DealCollectionName,
	)
	dealService := service.NewDeal(repository.NewDeal(dealCollection, getTestMarkets(), buildAvailableMarkets(conf)), getTestMarkets(), make(chan *model.Deal))
	avg, err := dealService.GetAvgPrice(ctx, time.Hour*24*40, "ETH_TRX")
	require.NoError(t, err)
	fmt.Println(avg)
}
