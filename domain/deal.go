package domain

import "time"

type Deal struct {
	ID     float64   `json:"_id"`
	Price  float64    `json:"price"`
	Volume float64     `json:"volume"`
	Time   time.Time `json:"time"`
	Market string    `json:"market"`
	DealId string    `json:"deal_id"`
}
