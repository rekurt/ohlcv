package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Candle struct {
	Symbol   string               `bson:"s"`
	Open     primitive.Decimal128 `bson:"o"`
	High     primitive.Decimal128 `bson:"h"`
	Low      primitive.Decimal128 `bson:"l"`
	Close    primitive.Decimal128 `bson:"c"`
	Volume   primitive.Decimal128 `bson:"v"`
	OpenTime time.Time            `bson:"t"`
}
