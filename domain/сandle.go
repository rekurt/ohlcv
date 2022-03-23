package domain

import "time"

type Candle struct {
	Open   string `json:"open"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Close  string `json:"close"`
	Volume float64 `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}
