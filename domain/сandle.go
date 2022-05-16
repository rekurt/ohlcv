package domain

import (
	"time"
)

type Candle struct {
	Symbol    string    `json:"symbol"`
	Open      string    `json:"o"`
	High      string    `json:"h"`
	Low       string    `json:"l"`
	Close     string    `json:"c"`
	Volume    string    `json:"v"`
	Timestamp time.Time `json:"t"`
}

type Chart struct {
	Symbol     string `json:"symbol"`
	resolution string
	O          []string `json:"o"`
	H          []string `json:"h"`
	L          []string `json:"l"`
	C          []string `json:"c"`
	V          []string `json:"v"`
	T          []int64  `json:"t"`
}

type ChartResponse struct {
	Symbol     string `json:"symbol"`
	resolution string
	O          []string `json:"o"`
	H          []string `json:"h"`
	L          []string `json:"l"`
	C          []string `json:"c"`
	V          []string `json:"v"`
	T          []int64  `json:"t"`
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
