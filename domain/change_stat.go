package domain

type TickerPriceChangeStatistics struct {
	Symbol             string
	PriceChange        string
	PriceChangePercent string
	WeightedAvgPrice   string
	PrevClosePrice     string
	LastPrice          string
	LastQty            string
	BidPrice           string
	BidQty             string
	AskPrice           string
	AskQty             string
	OpenPrice          string
	HighPrice          string
	LowPrice           string
	Volume             string
	QuoteVolume        string
	OpenTime           int64
	CloseTime          int64
	FirstId            string
	LastId             string
	Count              int
}
