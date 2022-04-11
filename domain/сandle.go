package domain

import "time"

const Candle1MInterval = "1m"
const Candle3MInterval = "3m"
const Candle5MInterval = "5m"
const Candle15MInterval = "15m"
const Candle30MInterval = "30m"
const Candle1HInterval = "1h"
const Candle2HInterval = "2h"
const Candle4HInterval = "4h"
const Candle6HInterval = "6h"
const Candle12HInterval = "12h"
const Candle1DInterval = "1d"
const Candle1MHInterval = "1mh"

type Candle struct {
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

type Chart struct {
	O []float64 `json:"o"`
	H []float64 `json:"h"`
	L []float64 `json:"l"`
	C []float64 `json:"c"`
	V []float64 `json:"v"`
	T []int64 `json:"t"`
}
