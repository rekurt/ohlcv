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

func StrIntervalToDuration(interval string) time.Duration {
	int2dur := map[string]time.Duration{
		Candle1MInterval:  time.Minute,
		Candle3MInterval:  3 * time.Minute,
		Candle5MInterval:  5 * time.Minute,
		Candle15MInterval: 15 * time.Minute,
		Candle30MInterval: 30 * time.Minute,
		Candle1HInterval:  60 * time.Minute,
		Candle2HInterval:  120 * time.Minute,
		Candle4HInterval:  240 * time.Minute,
		Candle6HInterval:  360 * time.Minute,
		Candle12HInterval: 720 * time.Minute,
		Candle1DInterval:  1440 * time.Minute,
		Candle1MHInterval: 43200 * time.Minute,
	}

	return int2dur[interval]
}
