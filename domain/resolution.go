package domain

import "time"

const (
	Day = 24 * time.Hour
)

type Resolution string

const (
	Candle1MResolution  Resolution = "1"
	Candle3MResolution  Resolution = "3"
	Candle5MResolution  Resolution = "5"
	Candle15MResolution Resolution = "15"
	Candle30MResolution Resolution = "30"
	Candle1HResolution  Resolution = "60"
	Candle2HResolution  Resolution = "120"
	Candle4HResolution  Resolution = "240"
	Candle6HResolution  Resolution = "360"
	Candle12HResolution Resolution = "720"
	Candle1MHResolution Resolution = "1MH"
	Candle1DResolution  Resolution = "1D"

	// LEGACY FOR BACKWARD COMPATIBILITY WITH OLD MOBILE APPS

	Candle1H2Resolution  Resolution = "1H"
	Candle2H2Resolution  Resolution = "2H"
	Candle4H2Resolution  Resolution = "4H"
	Candle6H2Resolution  Resolution = "6H"
	Candle12H2Resolution Resolution = "12H"
	Candle1WResolution   Resolution = "1W"
	Candle1MH2Resolution Resolution = "1M"
)

const MinuteUnit = "minute"
const HourUnit = "hour"
const DayUnit = "day"
const MonthUnit = "month"
const WeekUnit = "week"

func GetAvailableResolutions() []Resolution {
	return []Resolution{
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
		Candle1WResolution,
		Candle1MH2Resolution,
	}
}

func CalculateCloseTime(openTime time.Time, resolution Resolution) time.Time {
	duration := resolution.ToDuration(openTime.Month(), openTime.Year())

	return openTime.Add(duration - time.Nanosecond).UTC()
}

func (resolution Resolution) ToDuration(month time.Month, year int) time.Duration {
	int2dur := map[Resolution]time.Duration{
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
		Candle1MHResolution: monthDuration(month, year),
		// LEGACY FOR BACKWARD COMPATIBILITY WITH OLD MOBILE APPS
		Candle1H2Resolution:  60 * time.Minute,
		Candle2H2Resolution:  120 * time.Minute,
		Candle4H2Resolution:  240 * time.Minute,
		Candle6H2Resolution:  360 * time.Minute,
		Candle12H2Resolution: 720 * time.Minute,
		Candle1WResolution:   7 * Day,
		Candle1MH2Resolution: monthDuration(month, year),
	}

	if duration, ok := int2dur[resolution]; ok {
		return duration
	}

	return 0
}

func (resolution Resolution) IsNotExist() bool {
	resolutions := []Resolution{
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
		Candle1H2Resolution,
		Candle2H2Resolution,
		Candle4H2Resolution,
		Candle6H2Resolution,
		Candle12H2Resolution,
		Candle1WResolution,
		Candle1MH2Resolution,
	}

	for _, r := range resolutions {
		if resolution == r {
			return false
		}
	}

	return true
}

func monthDuration(month time.Month, year int) time.Duration {
	switch month {
	case time.January:
		return 31 * Day
	case time.February:
		if year%4 == 0 {
			return 29 * Day
		}
		return 28 * Day
	case time.March:
		return 31 * Day
	case time.April:
		return 30 * Day
	case time.May:
		return 31 * Day
	case time.June:
		return 30 * Day
	case time.July:
		return 31 * Day
	case time.August:
		return 31 * Day
	case time.September:
		return 30 * Day
	case time.October:
		return 31 * Day
	case time.November:
		return 30 * Day
	case time.December:
		return 31 * Day
	default:
		return 30 * Day
	}
}
