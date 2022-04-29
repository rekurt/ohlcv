package tests

import (
	mg "go.mongodb.org/mongo-driver/mongo"

	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"bitbucket.org/novatechnologies/ohlcv/infra/broker"
	"bitbucket.org/novatechnologies/ohlcv/infra/centrifuge"
)

func InitCandleService(
	conf infra.Config,
	dealsCollection *mg.Collection,
	eventsBroker domain.EventsBroker,
) *candle.Service {
	broadcaster := centrifuge.NewBroadcaster(
		centrifuge.NewPublisher(conf.CentrifugeConfig),
		eventsBroker,
	)
	broadcaster.SubscribeForCharts()

	return candle.NewService(
		&candle.Storage{DealsDbCollection: dealsCollection},
		new(candle.Agregator),
		domain.GetAvailableMarkets(),
		domain.GetAvailableResolutions(),
		broker.NewInMemory(),
	)
}
