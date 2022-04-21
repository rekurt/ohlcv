package domain

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Candle struct {
	Open      primitive.Decimal128   `json:"open"`
	High      primitive.Decimal128   `json:"high"`
	Low       primitive.Decimal128   `json:"low"`
	Close     primitive.Decimal128   `json:"close"`
	Volume    primitive.Decimal128   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

type Chart struct {
	market   string
	interval string
	O        []string `json:"o"`
	H        []string `json:"h"`
	L        []string `json:"l"`
	C        []string `json:"c"`
	V        []string `json:"v"`
	T        []int64   `json:"t"`
}

func (c *Chart) Interval() string {
	return c.interval
}

func (c *Chart) SetInterval(interval string) {
	c.interval = interval
}

func (c *Chart) Market() string {
	return c.market
}

func (c *Chart) SetMarket(market string) {
	c.market = market
}
