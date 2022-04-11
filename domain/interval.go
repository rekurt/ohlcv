package domain

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

func GetAvailableIntervals() []string {
	return []string{
		Candle1MInterval,
		Candle3MInterval,
		Candle5MInterval,
		Candle15MInterval,
		Candle30MInterval,
		Candle1HInterval,
		Candle2HInterval,
		Candle4HInterval,
		Candle6HInterval,
		Candle12HInterval,
		Candle1DInterval,
		Candle1MHInterval,
	}
}