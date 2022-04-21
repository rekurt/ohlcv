package domain

import "time"

const Candle1MInterval = "1"
const Candle3MInterval = "3"
const Candle5MInterval = "5"
const Candle15MInterval = "15"
const Candle30MInterval = "30"
const Candle1HInterval = "1H"
const Candle2HInterval = "2H"
const Candle4HInterval = "4H"
const Candle6HInterval = "6H"
const Candle12HInterval = "12H"
const Candle1DInterval = "1D"
const Candle1MHInterval = "1MH"

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
