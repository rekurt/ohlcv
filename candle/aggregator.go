package candle

import (
	"bitbucket.org/novatechnologies/ohlcv/internal/model"
	"context"
	"time"

	"bitbucket.org/novatechnologies/common/infra/logger"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"bitbucket.org/novatechnologies/ohlcv/domain"
)

type Aggregator struct{}

func (s Aggregator) AggregateCandleToChartByResolution(
	candles []*domain.Candle,
	market string,
	resolution model.Resolution,
	count int, // is not used. 0 for unlimit request
) *domain.Chart {
	var chart *domain.Chart

	if candles == nil {
		return &domain.Chart{}
	}
	logger.FromContext(context.Background()).WithField(
		"resolution",
		resolution,
	).Debugf("[CandleService] Call AggregateCandleToChartByResolution method.")
	switch resolution {
	case model.Candle1MResolution:
		chart = s.aggregateMinCandlesToChart(candles, market, 1, count)
	case model.Candle3MResolution:
		chart = s.aggregateMinCandlesToChart(candles, market, 3, count)
	case model.Candle5MResolution:
		chart = s.aggregateMinCandlesToChart(candles, market, 5, count)
	case model.Candle15MResolution:
		chart = s.aggregateMinCandlesToChart(candles, market, 15, count)
	case model.Candle30MResolution:
		chart = s.aggregateMinCandlesToChart(candles, market, 30, count)
	case model.Candle1HResolution,
		model.Candle1H2Resolution:
		chart = s.aggregateHoursCandlesToChart(candles, 1)
	case model.Candle2HResolution,
		model.Candle2H2Resolution:
		chart = s.aggregateHoursCandlesToChart(candles, 2)
	case model.Candle4HResolution,
		model.Candle4H2Resolution:
		chart = s.aggregateHoursCandlesToChart(candles, 4)
	case model.Candle6HResolution,
		model.Candle6H2Resolution:
		chart = s.aggregateHoursCandlesToChart(candles, 6)
	case model.Candle12HResolution,
		model.Candle12H2Resolution:
		chart = s.aggregateHoursCandlesToChart(candles, 12)
	case model.Candle1DResolution:
		chart = s.aggregateHoursCandlesToChart(candles, 24)
	case model.Candle1MHResolution,
		model.Candle1MH2Resolution:
		chart = s.aggregateMonthCandlesToChart(candles, market, count)
	case model.Candle1WResolution:
		chart = s.aggregateWeekCandlesToChart(candles)
	default:
		logger.FromContext(context.Background()).WithField(
			"resolution",
			resolution,
		).Errorf("Unsupported resolution.")

		return chart
	}

	chart.SetMarket(market)
	chart.SetResolution(resolution)

	return chart
}

func (s Aggregator) aggregateMinCandlesToChart(
	candles []*domain.Candle,
	market string,
	minute int,
	count int,
) *domain.Chart {
	result := make(map[int64]*domain.Candle)

	var min int
	var mod int
	var mul time.Duration
	var timestamp int64
	now := time.Now()
	currentTs := now.Add(time.Duration(now.Minute()%minute) * -time.Minute).Unix()
	for _, candle := range candles {
		var comparedCandle *domain.Candle
		min = int(int64(candle.OpenTime.Minute()))
		mod = min % minute
		mul = time.Duration(mod) * -time.Minute
		timestamp = candle.OpenTime.Add(mul).Unix()
		c := result[timestamp]

		if c != nil {
			comparedCandle = s.compare(c, candle)
		} else {
			comparedCandle = candle
		}

		result[timestamp] = comparedCandle
		if currentTs == timestamp {
			result[currentTs] = comparedCandle
		}
	}

	chart := s.GenerateChart(result)

	return chart
}

func (s Aggregator) compare(
	c *domain.Candle,
	candle *domain.Candle,
) *domain.Candle {
	comparedCandle := &domain.Candle{}
	if c.OpenTime.Unix() < candle.OpenTime.Unix() {
		comparedCandle.Open = c.Open
		comparedCandle.Close = candle.Close
	} else {
		comparedCandle.Open = candle.Open
		comparedCandle.Close = c.Close
	}

	cHight, _ := compareDecimal128(c.High, candle.High)
	if cHight == -1 {
		comparedCandle.High = candle.High
	}
	cLow, _ := compareDecimal128(c.Low, candle.Low)
	if cLow == 1 {
		comparedCandle.Low = candle.Low
	}
	dv1, _ := decimal.NewFromString(c.Volume.String())
	dv2, _ := decimal.NewFromString(candle.Volume.String())
	resultVolume, _ := primitive.ParseDecimal128(dv1.Add(dv2).String())
	comparedCandle.Volume = resultVolume
	comparedCandle.OpenTime = candle.OpenTime

	return comparedCandle
}

func (s *Aggregator) aggregateHoursCandlesToChart(candles []*domain.Candle, hour int) *domain.Chart {
	result := make(map[int64]*domain.Candle)

	var min int
	var mod int
	var mul time.Duration
	var timestamp int64
	for _, candle := range candles {
		min = int(int64(candle.OpenTime.Hour()))
		mod = min % hour
		mul = time.Duration(mod) * -time.Hour
		timestamp = candle.OpenTime.Add(mul).Truncate(time.Hour).Unix()
		c := result[timestamp]
		if c != nil {
			result[timestamp] = s.compare(c, candle)
		} else {
			result[timestamp] = candle
		}
	}

	chart := s.GenerateChart(result)

	return chart
}

func (s *Aggregator) aggregateMonthCandlesToChart(
	candles []*domain.Candle,
	market string,
	count int,
) *domain.Chart {
	result := make(map[int64]*domain.Candle)

	var timestamp int64
	for _, candle := range candles {
		timestamp = time.Date(
			candle.OpenTime.Year(),
			candle.OpenTime.Month(),
			1,
			0,
			0,
			0,
			0,
			time.Local,
		).Unix()
		c := result[timestamp]
		if c != nil {
			result[timestamp] = s.compare(c, candle)
		} else {
			result[timestamp] = candle
		}
	}

	chart := s.GenerateChart(result)

	return chart
}

func firstDayOfISOWeek(year int, week int, timezone *time.Location) time.Time {
	date := time.Date(year, 0, 0, 0, 0, 0, 0, timezone)
	isoYear, isoWeek := date.ISOWeek()
	for date.Weekday() != time.Monday { // iterate back to Monday
		date = date.AddDate(0, 0, -1)
		isoYear, isoWeek = date.ISOWeek()
	}
	for isoYear < year { // iterate forward to the first day of the first week
		date = date.AddDate(0, 0, 1)
		isoYear, isoWeek = date.ISOWeek()
	}
	for isoWeek < week { // iterate forward to the first day of the given week
		date = date.AddDate(0, 0, 1)
		isoYear, isoWeek = date.ISOWeek()
	}
	return date
}

func (s *Aggregator) aggregateWeekCandlesToChart(candles []*domain.Candle) *domain.Chart {
	result := make(map[int64]*domain.Candle)

	var timestamp int64
	for _, candle := range candles {
		year, week := candle.OpenTime.ISOWeek()
		timestamp = firstDayOfISOWeek(year, week, time.Local).Unix()
		c := result[timestamp]
		if c != nil {
			result[timestamp] = s.compare(c, candle)
		} else {
			result[timestamp] = candle
		}
	}

	chart := s.GenerateChart(result)

	return chart
}

func (s *Aggregator) GenerateChart(result map[int64]*domain.Candle) *domain.Chart {
	chart := &domain.Chart{
		O: make([]primitive.Decimal128, 0),
		H: make([]primitive.Decimal128, 0),
		L: make([]primitive.Decimal128, 0),
		C: make([]primitive.Decimal128, 0),
		V: make([]primitive.Decimal128, 0),
		T: make([]int64, 0),
	}

	for t, aggregatedCandle := range result {
		chart.O = append(chart.O, aggregatedCandle.Open)
		chart.H = append(chart.H, aggregatedCandle.High)
		chart.L = append(chart.L, aggregatedCandle.Low)
		chart.C = append(chart.C, aggregatedCandle.Close)
		chart.V = append(chart.V, aggregatedCandle.Volume)
		chart.T = append(chart.T, t)
	}

	return chart
}

func compareDecimal128(d1, d2 primitive.Decimal128) (int, error) {
	b1, exp1, err := d1.BigInt()
	if err != nil {
		return 0, err
	}
	b2, exp2, err := d2.BigInt()
	if err != nil {
		return 0, err
	}

	sign := b1.Sign()
	if sign != b2.Sign() {
		if b1.Sign() > 0 {
			return 1, nil
		} else {
			return -1, nil
		}
	}

	if exp1 == exp2 {
		return b1.Cmp(b2), nil
	}

	if sign < 0 {
		if exp1 < exp2 {
			return 1, nil
		}
		return -1, nil
	} else {
		if exp1 < exp2 {
			return -1, nil
		}

		return 1, nil
	}
}

func addPrimitiveDecimal128(a, b primitive.Decimal128) (primitive.Decimal128, error) {
	ad, err := decimal.NewFromString(a.String())
	if err != nil {
		return primitive.Decimal128{}, err
	}
	bd, err := decimal.NewFromString(b.String())
	if err != nil {
		return primitive.Decimal128{}, err
	}
	result, err := primitive.ParseDecimal128(ad.Add(bd).String())
	if err != nil {
		return primitive.Decimal128{}, err
	}
	return result, nil
}

func (s *Aggregator) GetResolutionStartTimestampByTime(resolution model.Resolution, time time.Time) int64 {
	var ts int64
	switch resolution {
	case model.Candle1MResolution:
		ts = getStartMinuteTs(time, 1)
	case model.Candle3MResolution:
		ts = getStartMinuteTs(time, 3)
	case model.Candle5MResolution:
		ts = getStartMinuteTs(time, 5)
	case model.Candle15MResolution:
		ts = getStartMinuteTs(time, 15)
	case model.Candle30MResolution:
		ts = getStartMinuteTs(time, 30)
	case model.Candle1HResolution:
		ts = getStartHourTs(time, 1)
	case model.Candle1H2Resolution:
		ts = getStartHourTs(time, 1)
	case model.Candle2HResolution:
		ts = getStartHourTs(time, 2)
	case model.Candle2H2Resolution:
		ts = getStartHourTs(time, 2)
	case model.Candle4HResolution:
		ts = getStartHourTs(time, 4)
	case model.Candle4H2Resolution:
		ts = getStartHourTs(time, 4)
	case model.Candle6HResolution:
		ts = getStartHourTs(time, 6)
	case model.Candle6H2Resolution:
		ts = getStartHourTs(time, 6)
	case model.Candle12HResolution:
		ts = getStartHourTs(time, 12)
	case model.Candle12H2Resolution:
		ts = getStartHourTs(time, 12)
	case model.Candle1DResolution:
		ts = getStartHourTs(time, 24)
	case model.Candle1MHResolution:
		ts = getStartMonthTs(time, 1)
	case model.Candle1MH2Resolution:
		ts = getStartMonthTs(time, 1)
	case model.Candle1WResolution:
		ts = getStartWeekTs(time)
	default:
		logger.FromContext(context.Background()).WithField(
			"resolution",
			resolution,
		).Errorf("Unsupported resolution.")
	}

	return ts
}

func getStartWeekTs(t time.Time) int64 {
	year, week := t.ISOWeek()
	return firstDayOfISOWeek(year, week, time.Local).Unix()
}

func getStartMinuteTs(candleTime time.Time, minute int) int64 {
	currentTs := candleTime.Add(time.Duration(candleTime.Minute()%minute) * -time.Minute).Truncate(time.Minute).Unix()

	return currentTs
}

func getStartHourTs(candleTime time.Time, h int) int64 {
	currentTs := candleTime.Add(time.Duration(candleTime.Hour()%h) * -time.Hour).Truncate(time.Hour).Unix()

	return currentTs
}

func getStartMonthTs(candleTime time.Time, m int) int64 {
	currentTs := time.Date(
		candleTime.Year(),
		candleTime.Month(),
		1,
		0,
		0,
		0,
		0,
		time.Local,
	).Unix()

	return currentTs
}
