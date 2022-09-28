package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Kline struct {
	OpenTime    time.Time            `bson:"openTime"`
	Open        primitive.Decimal128 `bson:"open"`
	High        primitive.Decimal128 `bson:"high"`
	Low         primitive.Decimal128 `bson:"low"`
	Close       primitive.Decimal128 `bson:"close"`
	Volume      primitive.Decimal128 `bson:"volume"`
	CloseTime   time.Time            `bson:"closeTime"`
	Quotes      primitive.Decimal128 `bson:"quotes"`
	Trades      int                  `bson:"trades"`
	TakerAssets primitive.Decimal128 `bson:"takerAssets"`
	TakerQuotes primitive.Decimal128 `bson:"takerQuotes"`
	Symbol      string               `bson:"symbol"`
	First       time.Time            `bson:"first"`
	Last        time.Time            `bson:"last"`
}
