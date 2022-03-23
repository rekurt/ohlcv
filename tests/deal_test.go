package tests

import (
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"bitbucket.org/novatechnologies/ohlcv/api/http"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/deal"
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
	dealService := deal.NewService(dealCollection)
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

	if err != nil {
		t.Failed()
	}

	d2 := matcher.Deal{
		Id:           "1234567",
		Market:       market,
		MakerOrderId: "12345",
		TakerOrderId: "12345",
		CreatedAt:    time.Now().Unix(),
		Price:        "52.300",
		Amount:       "0.0121",
	}
	_, err = dealService.SaveDeal(ctx, d2)

	if err != nil {
		t.Fail()
	}

	candles, _ := candle.NewService(dealCollection, getTestMarkets()).GetMinuteCandles(ctx, market)
	//res := candles[len(candles)-1]
	candle, _ := candle.NewService(dealCollection, getTestMarkets()).GetLastCandle(ctx, market)

	log.Print(candles, candle)
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

func getTestMarkets() []string {
	return []string{
		"BTC-USDT",
	}
}
