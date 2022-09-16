package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Kline struct {
	OpenTime    time.Time            `json:"openTime"`
	Open        primitive.Decimal128 `json:"open"`
	High        primitive.Decimal128 `json:"high"`
	Low         primitive.Decimal128 `json:"low"`
	Close       primitive.Decimal128 `json:"close"`
	Volume      primitive.Decimal128 `json:"volume"`
	CloseTime   time.Time            `json:"closeTime"`
	Quote       primitive.Decimal128 `json:"quote"`
	Trades      int                  `json:"trades"`
	TakerAssets primitive.Decimal128 `json:"takerAssets"`
	TakerQuotes primitive.Decimal128 `json:"takerQuotes"`
}
