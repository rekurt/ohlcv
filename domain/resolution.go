package domain

import "time"

const (
	Candle1MResolution  = "1"
	Candle3MResolution  = "3"
	Candle5MResolution  = "5"
	Candle15MResolution = "15"
	Candle30MResolution = "30"
	Candle1HResolution  = "60"
	Candle2HResolution  = "120"
	Candle4HResolution  = "240"
	Candle6HResolution  = "360"
	Candle12HResolution = "720"
	Candle1MHResolution = "1MH"
	Candle1DResolution  = "1D"

	// LEGACY FOR BACKWARD COMPATIBILITY WITH OLD MOBILE APPS

	Candle1H2Resolution  = "1H"
	Candle2H2Resolution  = "2H"
	Candle4H2Resolution  = "4H"
	Candle6H2Resolution  = "6H"
	Candle12H2Resolution = "12H"
	Candle1MH2Resolution = "1M"
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

		// LEGACY FOR BACKWARD COMPATIBILITY WITH OLD MOBILE APPS

		Candle1H2Resolution,
		Candle2H2Resolution,
		Candle4H2Resolution,
		Candle6H2Resolution,
		Candle12H2Resolution,
		Candle1MH2Resolution,
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
		// LEGACY FOR BACKWARD COMPATIBILITY WITH OLD MOBILE APPS
		Candle1H2Resolution:  60 * time.Minute,
		Candle2H2Resolution:  120 * time.Minute,
		Candle4H2Resolution:  240 * time.Minute,
		Candle6H2Resolution:  360 * time.Minute,
		Candle12H2Resolution: 720 * time.Minute,
		Candle1MH2Resolution: 43200 * time.Minute,
	}

	return int2dur[resolution]
}
