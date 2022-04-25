package candle

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"

	"bitbucket.org/novatechnologies/common/infra/logger"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"bitbucket.org/novatechnologies/ohlcv/domain"
)

type Service struct {
	DealsDbCollection  *mongo.Collection
	Markets            map[string]string
	UpdatedCandles     chan *domain.Chart
	AvailableIntervals []string
}

const CurrentTimestamp = "current"

func NewService(
	dealsDbCollection *mongo.Collection,
	markets map[string]string,
	availableIntervals []string,
) *Service {
	c := make(chan *domain.Chart, 2000)
	return &Service{
		DealsDbCollection:  dealsDbCollection,
		Markets:            markets,
		UpdatedCandles:     c,
		AvailableIntervals: availableIntervals,
	}
}

//Generation for websocket pushing new candle every min (example: empty candles)
func (s *Service) CronCandleGenerationStart(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		done := make(chan bool)
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				for _, market := range s.Markets {
					logger.FromContext(ctx).WithField(
						"market",
						market,
					).Infof("[CronCandleGenerationStart]Getting new candle for the market.")
					// s.PushUpdatedCandleEvent(ctx, market)
				}
			}
		}
	}()
}

func (s Service) GetMinuteCandles(
	ctx context.Context,
	market string,
	period ...time.Time,
) ([]*domain.Candle, error) {
	logger.FromContext(ctx).WithField(
		"market",
		market,
	).Infof("[CandleService] Call GetMinuteCandles")
	from, to := period[0], period[1]

	matchStage := bson.D{
		{"$match", bson.D{
			{"market", market},
			{"time", bson.D{
				{"$gt", primitive.NewDateTimeFromTime(from)},
				{"$lt", primitive.NewDateTimeFromTime(to)},
			}},
		}},
	}

	groupStage := bson.D{{"$group", bson.D{
		{"_id", bson.D{
			{"market", "$market"},
			{"time", bson.D{
				{" $dateTrunc", bson.D{
					{"date", "$time"},
					{"unit", "$minute"},
					{"binSize", 1},
				}},
			}},
		}},
		{"high", bson.D{{"$max", "$price"}}},
		{"low", bson.D{{"$min", "$price"}}},
		{"open", bson.D{{"$first", "$price"}}},
		{"close", bson.D{{"$last", "$price"}}},
		{"volume", bson.D{{"$sum", "$volume"}}},
		{"timestamp", bson.D{{"$first", "$time"}}},
	}}}

	sortStage := bson.D{{"$sort", bson.D{
		{
			"_id.time", 1,
		},
	}}}

	options := options.Aggregate()
	adu := true
	options.AllowDiskUse = &adu
	cursor, err := s.DealsDbCollection.Aggregate(
		ctx,
		mongo.Pipeline{matchStage, groupStage, sortStage},
		options,
	)

	if err != nil {
		logger.FromContext(ctx).WithField(
			"error",
			err,
		).Errorf("[CandleService]Failed apply a aggregation function on the collection.")
		return nil, err
	}

	_ = make([]*domain.Deal, 0)
	candles := make([]*domain.Candle, 0)

	err = cursor.All(ctx, &candles)
	log.Print(candles)

	if err != nil {
		logger.FromContext(ctx).WithField(
			"error",
			err,
		).Errorf("[CandleService]Failed apply a aggregation function on the collection.")
		return nil, err
	}

	if len(candles) == 0 {
		logger.FromContext(ctx).WithField(
			"candleCount",
			len(candles),
		).WithField("err", err).WithField("err", period).Infof("Candles not found.")
		return nil, err
	}
	logger.FromContext(ctx).WithField(
		"candleCount",
		len(candles),
	).Infof("Success get candles.")

	return candles, err
}

func (s Service) GetCurrentCandle(
	ctx context.Context,
	market string,
	interval string,
) (*domain.Chart, error) {
	candleDuration := domain.StrIntervalToDuration(interval)
	from := time.Now().Add(-candleDuration).Truncate(candleDuration)
	to := time.Now()

	cs, err := s.GetMinuteCandles(ctx, market, from, to)
	chart := s.AggregateCandleToChartByInterval(cs, market, interval, 1)
	chart.SetMarket(market)
	chart.SetInterval(interval)

	return chart, err
}

func (s Service) PushLastUpdatedCandle(
	ctx context.Context,
	market string,
	interval string,
) {
	logger.FromContext(context.Background()).
		WithField("interval", interval).
		WithField("market", market).
		Infof("[CandleService] Call PushLastUpdatedCandle method.")
	upd, _ := s.GetCurrentCandle(ctx, market, interval)
	if upd != nil {
		//s.UpdatedCandles <- upd
	}

}

func (s Service) AggregateCandleToChartByInterval(
	candles []*domain.Candle,
	market string,
	interval string,
	count int, //is not used. 0 for unlimit request
) *domain.Chart {
	var chart *domain.Chart

	logger.FromContext(context.Background()).WithField(
		"interval",
		interval,
	).Infof("[CandleService] Call AggregateCandleToChartByInterval method.")
	switch interval {
	case domain.Candle1MInterval:
		chart = s.aggregateMinCandlesToChart(candles, market, 1, count)
	case domain.Candle3MInterval:
		chart = s.aggregateMinCandlesToChart(candles, market, 3, count)
	case domain.Candle5MInterval:
		chart = s.aggregateMinCandlesToChart(candles, market, 5, count)
	case domain.Candle15MInterval:
		chart = s.aggregateMinCandlesToChart(candles, market, 15, count)
	case domain.Candle30MInterval:
		chart = s.aggregateMinCandlesToChart(candles, market, 30, count)
	case domain.Candle1HInterval:
		chart = s.aggregateHoursCandlesToChart(candles, market, 1, count)
	case domain.Candle2HInterval:
		chart = s.aggregateHoursCandlesToChart(candles, market, 2, count)
	case domain.Candle4HInterval:
		chart = s.aggregateHoursCandlesToChart(candles, market, 4, count)
	case domain.Candle6HInterval:
		chart = s.aggregateHoursCandlesToChart(candles, market, 6, count)
	case domain.Candle12HInterval:
		chart = s.aggregateHoursCandlesToChart(candles, market, 12, count)
	case domain.Candle1DInterval:
		chart = s.aggregateHoursCandlesToChart(candles, market, 24, count)
	case domain.Candle1MHInterval:
		chart = s.aggregateMonthCandlesToChart(candles, market, count)
	default:
		logger.FromContext(context.Background()).WithField(
			"interval",
			interval,
		).Errorf("Unsupported interval.")
	}

	return chart
}

func (s Service) aggregateMinCandlesToChart(candles []*domain.Candle, market string, minute int, count int) *domain.Chart {
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

func (s Service) compare(
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

	cHight, _ := CompareDecimal128(c.High, candle.High)
	if  cHight == -1 {
		comparedCandle.High = candle.High
	}
	cLow, _ := CompareDecimal128(c.Low, candle.Low)
	if cLow == 1{
		comparedCandle.Low = candle.Low
	}
	dv1, _ := decimal.NewFromString(c.Volume.String())
	dv2, _ := decimal.NewFromString(candle.Volume.String())
	resultVolume, _ :=  primitive.ParseDecimal128(dv1.Add(dv2).String())
	comparedCandle.Volume = resultVolume
	comparedCandle.Timestamp = candle.Timestamp

	return comparedCandle
}

func (s *Service) aggregateHoursCandlesToChart(candles []*domain.Candle, market string, hour int, count int, ) *domain.Chart {
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

func (s *Service) aggregateMonthCandlesToChart(candles []*domain.Candle, market string, count int, ) *domain.Chart {
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

func (s *Service) GenerateChart(result map[int64]*domain.Candle) *domain.Chart {
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

func (s *Service) PushUpdatedCandleEvent(ctx context.Context, market string) {
	for _, interval := range s.AvailableIntervals {
		s.PushLastUpdatedCandle(ctx, market, interval)
	}
}

func CompareDecimal128(d1, d2 primitive.Decimal128) (int, error) {
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
