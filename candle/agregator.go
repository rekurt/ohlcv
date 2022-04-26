package candle

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"context"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Agregator struct {
}

func (s Agregator) AggregateCandleToChartByResolution(
	candles []*domain.Candle,
	market string,
	resolution string,
	count int, //is not used. 0 for unlimit request
) *domain.Chart {
	var chart *domain.Chart

	logger.FromContext(context.Background()).WithField(
		"resolution",
		resolution,
	).Infof("[CandleService] Call AggregateCandleToChartByResolution method.")
	switch resolution {
	case domain.Candle1MResolution:
		chart = s.aggregateMinCandlesToChart(candles, market, 1, count)
	case domain.Candle3MResolution:
		chart = s.aggregateMinCandlesToChart(candles, market, 3, count)
	case domain.Candle5MResolution:
		chart = s.aggregateMinCandlesToChart(candles, market, 5, count)
	case domain.Candle15MResolution:
		chart = s.aggregateMinCandlesToChart(candles, market, 15, count)
	case domain.Candle30MResolution:
		chart = s.aggregateMinCandlesToChart(candles, market, 30, count)
	case domain.Candle1HResolution:
		chart = s.aggregateHoursCandlesToChart(candles, market, 1, count)
	case domain.Candle2HResolution:
		chart = s.aggregateHoursCandlesToChart(candles, market, 2, count)
	case domain.Candle4HResolution:
		chart = s.aggregateHoursCandlesToChart(candles, market, 4, count)
	case domain.Candle6HResolution:
		chart = s.aggregateHoursCandlesToChart(candles, market, 6, count)
	case domain.Candle12HResolution:
		chart = s.aggregateHoursCandlesToChart(candles, market, 12, count)
	case domain.Candle1DResolution:
		chart = s.aggregateHoursCandlesToChart(candles, market, 24, count)
	case domain.Candle1MHResolution:
		chart = s.aggregateMonthCandlesToChart(candles, market, count)
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

func (s Agregator) aggregateMinCandlesToChart(candles []*domain.Candle, market string, minute int, count int) *domain.Chart {
	result := make(map[int64]*domain.Candle)

	var min int
	var mod int
	var mul time.Duration
	var timestamp int64
	now := time.Now()
	currentTs := now.Add(time.Duration(now.Minute()%minute) * -time.Minute).Unix()
	for _, candle := range candles {
		var comparedCandle *domain.Candle
		min = int(int64(candle.Timestamp.Minute()))
		mod = min % minute
		mul = time.Duration(mod) * -time.Minute
		timestamp = candle.Timestamp.Add(mul).Unix()
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

func (s Agregator) compare(
	c *domain.Candle,
	candle *domain.Candle,
) *domain.Candle {
	comparedCandle := &domain.Candle{}
	if c.Timestamp.Unix() < candle.Timestamp.Unix() {
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
	comparedCandle.Timestamp = candle.Timestamp

	return comparedCandle
}

func (s *Agregator) aggregateHoursCandlesToChart(candles []*domain.Candle, market string, hour int, count int, ) *domain.Chart {
	result := make(map[int64]*domain.Candle)

	var min int
	var mod int
	var mul time.Duration
	var timestamp int64
	for _, candle := range candles {
		min = int(int64(candle.Timestamp.Hour()))
		mod = min % hour
		mul = time.Duration(mod) * -time.Hour
		timestamp = candle.Timestamp.Add(mul).Unix()
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

func (s *Agregator) aggregateMonthCandlesToChart(candles []*domain.Candle, market string, count int, ) *domain.Chart {
	result := make(map[int64]*domain.Candle)

	var timestamp int64
	for _, candle := range candles {
		timestamp = time.Date(
			candle.Timestamp.Year(),
			candle.Timestamp.Month(),
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

func (s *Agregator) GenerateChart(result map[int64]*domain.Candle) *domain.Chart {
	chart := &domain.Chart{
		O: make([]string, 0),
		H: make([]string, 0),
		L: make([]string, 0),
		C: make([]string, 0),
		V: make([]string, 0),
		T: make([]int64, 0),
	}

	for t, aggregatedCandle := range result {
		chart.O = append(chart.O, aggregatedCandle.Open.String())
		chart.H = append(chart.H, aggregatedCandle.High.String())
		chart.L = append(chart.L, aggregatedCandle.Low.String())
		chart.C = append(chart.C, aggregatedCandle.Close.String())
		chart.V = append(chart.V, aggregatedCandle.Volume.String())
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