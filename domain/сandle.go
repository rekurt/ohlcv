package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Candle struct {
	Symbol    string               `json:"symbol"`
	Open      primitive.Decimal128 `json:"o"`
	High      primitive.Decimal128 `json:"h"`
	Low       primitive.Decimal128 `json:"l"`
	Close     primitive.Decimal128 `json:"c"`
	Volume    primitive.Decimal128 `json:"v"`
	Timestamp time.Time            `json:"t"`
}

type Chart struct {
	Symbol     string `json:"symbol"`
	resolution string
	O          []primitive.Decimal128 `json:"o"`
	H          []primitive.Decimal128 `json:"h"`
	L          []primitive.Decimal128 `json:"l"`
	C          []primitive.Decimal128 `json:"c"`
	V          []primitive.Decimal128 `json:"v"`
	T          []int64                `json:"t"`
}

func (c *Chart) Resolution() string {
	return c.resolution
}

func (c *Chart) SetResolution(resolution string) {
	c.resolution = resolution
}

func (c *Chart) Market() string {
	return c.Symbol
}

func (c *Chart) SetMarket(market string) {
	c.Symbol = market
}
