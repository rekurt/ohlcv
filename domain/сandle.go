package domain

import "time"

type Candle struct {
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

type Chart struct {
	market   string
	interval string
	O        []float64 `json:"o"`
	H        []float64 `json:"h"`
	L        []float64 `json:"l"`
	C        []float64 `json:"c"`
	V        []float64 `json:"v"`
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
