package http

import (
	"os"
	"os/signal"
	"testing"

	"bitbucket.org/novatechnologies/ohlcv/infra"
	"bitbucket.org/novatechnologies/ohlcv/infra/broker"
	"bitbucket.org/novatechnologies/ohlcv/infra/mongo"
	"bitbucket.org/novatechnologies/ohlcv/internal/repository"
	"bitbucket.org/novatechnologies/ohlcv/tests"
)

func Test_Server_manual(t *testing.T) {
	t.Skip()
	ctx := infra.GetContext()
	conf := infra.SetConfig("../../config/.env")

	eventsBroker := broker.NewInMemory()
	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)
	dealCollection := mongo.GetCollection(
		ctx,
		mongoDbClient,
		conf.MongoDbConfig,
		conf.MongoDbConfig.DealCollectionName,
	)

	dealService := repository.NewDeal(dealCollection, tests.GetAvailableMarkets(), nil)
	candleService := tests.InitCandleService(conf, dealCollection, eventsBroker)

	server := NewServer(candleService, dealService, conf)
	server.Start(ctx)

	// shutdown
	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, os.Interrupt)

	_ = <-signalCh

	server.Stop(ctx)
}
