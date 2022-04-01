package tests

import (
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"bitbucket.org/novatechnologies/ohlcv/api/http"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/deal"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"bitbucket.org/novatechnologies/ohlcv/infra/mongo"
	"log"
	"os"
	"os/signal"
	"testing"
	"time"
)

func TestSaveDeal(t *testing.T) {
	ctx := infra.GetContext()
	conf := infra.SetConfig(ctx, "../config/.env")

	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)
	//mongo.InitDealCollection(ctx, mongoDbClient, conf.MongoDbConfig)
	dealCollection := mongo.GetCollection(ctx, mongoDbClient, conf.MongoDbConfig)
	dealService := deal.NewService(dealCollection, getTestMarkets())
	market := "BTC-USDT"

	d1 := matcher.Deal{
		Id:           "1234567",
		Market:       market,
		MakerOrderId: "12345",
		TakerOrderId: "12345",
		CreatedAt:    time.Now().Unix(),
		Price:        "102.300",
		Amount:       "0.0031",
	}
	_, err := dealService.SaveDeal(ctx, d1)

	d2 := matcher.Deal{
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

	d3 := matcher.Deal{
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

	candles, _ := candle.NewService(dealCollection, getTestMarkets()).GetMinuteCandles(ctx, market)
	chart5Min := candle.NewService(dealCollection, getTestMarkets()).AggregateCandleToChartByInterval(candles, domain.Candle5MInterval, 0)
	//res := candles[len(candles)-1]
	candle, _ := candle.NewService(dealCollection, getTestMarkets()).GetLastCandle(ctx, market, domain.Candle5MInterval)
	//assert.Equal(t, a, b, "The two words should be the same.")
	log.Print(candles, candle, chart5Min)
}

func TestDealGenerator(t *testing.T) {
	ctx := infra.GetContext()
	conf := infra.SetConfig(ctx, "../config/.env")

	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)
	//mongo.InitDealCollection(ctx, mongoDbClient, conf.MongoDbConfig)
	dealCollection := mongo.GetCollection(ctx, mongoDbClient, conf.MongoDbConfig)
	candleService := candle.NewService(dealCollection, getTestMarkets())

	candleService.CronCandleGenerationStart(ctx)

	server := http.NewServer(candleService)
	server.Start(ctx)

	//shutdown
	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, os.Interrupt)

	_ = <-signalCh
}

func getTestMarkets() map[string]string {
	return map[string]string{
		"BTC-USDT":"BTC-USDT",
	}
}
