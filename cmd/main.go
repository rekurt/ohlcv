package main

import (
	"bitbucket.org/novatechnologies/common/events/topics"
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"bitbucket.org/novatechnologies/ohlcv/api/http"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/deal"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"bitbucket.org/novatechnologies/ohlcv/infra/mongo"
	"context"
	"google.golang.org/protobuf/proto"
	"os"
	"os/signal"
)

func main() {
	ctx := infra.GetContext()
	conf := infra.SetConfig(ctx, "./config/.env")

	consumer := infra.NewConsumer(ctx, conf.KafkaConfig)

	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)
	dealCollection := mongo.GetCollection(ctx, mongoDbClient, conf.MongoDbConfig)
	dealService := deal.NewService(dealCollection, getMarkets())
	candleService := candle.NewService(dealCollection, getMarkets())

	server := http.NewServer(candleService)
	server.Start(ctx)

	go func() {
		consumer.Consume(ctx, topics.MatcherMDDeals, func(ctx context.Context, msg []byte) error {

			dealMessage := matcher.Deal{}
			if er := proto.Unmarshal(msg, &dealMessage); er != nil {
				logger.FromContext(ctx).WithField("method", "consumer.deals.Unmarshal").Errorf("%v", er)
				os.Exit(1)
			}

			d, _ := dealService.SaveDeal(ctx, dealMessage)
			println(d.Market)
			//candleService.PushLastUpdatedCandle(ctx, dealMessage.Market, domain.Candle1MInterval)
			//candleService.PushLastUpdatedCandle(ctx, dealMessage.Market, domain.Candle5MInterval)
			return nil
		})
	}()

	candleService.CronCandleGenerationStart(ctx)


	//shutdown
	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, os.Interrupt)

	_ = <-signalCh

	server.Stop(ctx)

	return
}

func getMarkets() map[string]string {
	return map[string]string{
		"af8327a8-2599-44b9-9813-8fb3dd236fb0":"USDT_TRX",
		"7ea7fbf5-cd49-4de7-a432-9806004e018d":"USDT_ETH",
		"17f0f56f-7979-4a21-99df-f74f25ac56d4":"USDT_LTC",
		"15954774-df52-4393-bb87-95b1f1e149e3":"USDT_BCH",
		"42e98b73-ec0c-4185-b0db-ffc8610f5741":"BTC_XRP",
		"352656ec-4ad4-4e8b-8dc4-2ddd3e7643b1":"USDT_BTC",
		"8177b9c1-dd10-49b6-800a-235e429c97dd":"BTC_BCH",
		"18da3c5f-7fa2-41cd-b053-00727fececdc":"ETH_TRX",
		"b8e3bfce-0b1e-4eb3-9b62-e0fd9b80ada4":"BTC_ETH",
		"e9a4ed4e-75cb-43c5-9f61-8bd97b63fb23":"BTC_LINK",
		"637549dd-48a6-4817-8d7b-2c0428dab380":"USDT_LINK",
		"e269c058-3a86-481b-87a3-85fad2d0c74d":"ETH_LINK",
		"86c3be16-b5ce-4b12-8704-80a2b283a26e":"ETH_LTC",
		"72793962-bab5-41a4-9c86-ad79ff984d2d":"BTC_LTC",
		"6229966b-0e92-4c00-acd7-e5f827cfed05":"ETH_XRP",
		"f59eecfd-db38-4f29-b854-e869e056b7d9":"BTC_TRX",
	}
}
