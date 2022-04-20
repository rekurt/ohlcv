package domain

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Deal struct {
	ID           primitive.ObjectID   `json:"_id"`
	Price        primitive.Decimal128 `json:"price"`
	Volume       primitive.Decimal128 `json:"volume"`
	Time         time.Time            `json:"time"`
	Market       string               `json:"market"`
	DealId       string               `json:"dealId"`
	IsBuyerMaker bool                 `json:"isBuyerMaker"`
}
