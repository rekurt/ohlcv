package http

import (
	"os"
	"os/signal"
	"testing"

	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/deal"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"bitbucket.org/novatechnologies/ohlcv/infra/mongo"
)

func Test_Server_manual(t *testing.T) {
	t.Skip()
	ctx := infra.GetContext()
	conf := infra.SetConfig("../../config/.env")

	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)
	dealCollection := mongo.GetCollection(ctx, mongoDbClient, conf.MongoDbConfig)

	dealService := deal.NewService(dealCollection, domain.GetAvailableMarkets())
	candleService := candle.NewService(dealCollection, domain.GetAvailableMarkets(), domain.GetAvailableResolutions())

	server := NewServer(candleService, dealService, nil)
	server.Start(ctx)

	// shutdown
	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, os.Interrupt)

	_ = <-signalCh

	server.Stop(ctx)
}
