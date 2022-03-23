package tests

import (
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/deal"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"bitbucket.org/novatechnologies/ohlcv/infra/mongo"
	"log"
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
	timeValue := time.Now()
	market := "BTC-USDT"
	d1 := &domain.Deal{
		Price:  "102.300",
		Volume: 0.0031,
		DealId: "1234567",
		Market: "BTC-USDT",
		Time:   timeValue,
	}
	_, err := dealService.SaveDeal(ctx, d1)

	if err != nil {
		t.Failed()
	}

	duplicate := &domain.Deal{
		Price:  "56.112",
		Volume: 0.0039,
		DealId: "1234567",
		Market: "BTC-USDT",
		Time:   timeValue,
	}
	_, err = dealService.SaveDeal(ctx, duplicate)

	if err != nil {
		t.Fail()
	}

	candles, _ := candle.NewService(dealCollection).GetMinuteCandles(ctx, market)
	candle, _ := candle.NewService(dealCollection).GetLastCandle(ctx, market)

	log.Print(candles, candle)
}
