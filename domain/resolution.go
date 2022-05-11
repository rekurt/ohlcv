package domain

import "time"

const (
	Candle1MResolution  = "1"
	Candle3MResolution  = "3"
	Candle5MResolution  = "5"
	Candle15MResolution = "15"
	Candle30MResolution = "30"
	Candle1HResolution  = "1H"
	Candle2HResolution  = "2H"
	Candle4HResolution  = "4H"
	Candle6HResolution  = "6H"
	Candle12HResolution = "12H"
	Candle1DResolution  = "1D"
	Candle1MHResolution = "1MH"
)

const MinuteUnit = "minute"
const HourUnit = "hour"
const DayUnit = "day"
const MonthUnit = "month"

func GetAvailableResolutions() []string {
	return []string{
		Candle1MResolution,
		Candle3MResolution,
		Candle5MResolution,
		Candle15MResolution,
		Candle30MResolution,
		Candle1HResolution,
		Candle2HResolution,
		Candle4HResolution,
		Candle6HResolution,
		Candle12HResolution,
		Candle1DResolution,
		Candle1MHResolution,
	}
}

func StrResolutionToDuration(resolution string) time.Duration {
	int2dur := map[string]time.Duration{
		Candle1MResolution:  time.Minute,
		Candle3MResolution:  3 * time.Minute,
		Candle5MResolution:  5 * time.Minute,
		Candle15MResolution: 15 * time.Minute,
		Candle30MResolution: 30 * time.Minute,
		Candle1HResolution:  60 * time.Minute,
		Candle2HResolution:  120 * time.Minute,
		Candle4HResolution:  240 * time.Minute,
		Candle6HResolution:  360 * time.Minute,
		Candle12HResolution: 720 * time.Minute,
		Candle1DResolution:  1440 * time.Minute,
		Candle1MHResolution: 43200 * time.Minute,
	}

	return int2dur[resolution]
}
