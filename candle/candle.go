package candle

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type Service struct {
	DealsDbCollection *mongo.Collection
	Markets           map[string]string
	UpdatedCandles    chan *domain.Chart
}

func NewService(dealsDbCollection *mongo.Collection, markets map[string]string) *Service {
	c := make(chan *domain.Chart, 100)
	return &Service{
		DealsDbCollection: dealsDbCollection,
		Markets:           markets,
		UpdatedCandles:    c,
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
					logger.FromContext(ctx).WithField("market", market).Infof("[CronCandleGenerationStart]Getting new candle for the market.")
					s.PushLastUpdatedCandle(ctx, market, domain.Candle5MInterval)
				}
			}
		}
	}()
}

func (s Service) GetMinuteCandles(ctx context.Context, market string) ([]*domain.Candle, error) {
	logger.FromContext(ctx).WithField("market", market).Infof("[CandleService] Call GetMinuteCandles")
	matchStage := bson.D{{"$match", bson.D{{"market", market}}}}
	groupStage := bson.D{{"$group", bson.D{
		{"_id", bson.D{
			{"market", "$market"},
			{"time", bson.D{
				{"date", "$time"},
				{"unit", "$minute"},
				{"binSize", 5},
			}},
		}},
		{"high", bson.D{{"$max", "$price"}}},
		{"low", bson.D{{"$min", "$price"}}},
		{"open", bson.D{{"$first", "$price"}}},
		{"close", bson.D{{"$last", "$price"}}},
		{"volume", bson.D{{"$sum", "$volume"}}},
		{"timestamp", bson.D{{"$first", "$time"}}},
	},
	}}

	sortStage := bson.D{{"$sort", bson.D{
		{
			"_id.time", 1,
		},
	}}}
	cursor, err := s.DealsDbCollection.Aggregate(ctx, mongo.Pipeline{matchStage, groupStage, sortStage})

	if err != nil {
		logger.FromContext(ctx).WithField("error", err).Errorf("[CandleService]Failed apply a aggregation function on the collection.")
		return nil, err
	}

	var candles []*domain.Candle
	err = cursor.All(ctx, &candles)
	if err != nil {
		logger.FromContext(ctx).WithField("error", err).Errorf("[CandleService]Failed apply a aggregation function on the collection.")
		return nil, err
	}

	if len(candles) == 0 {
		logger.FromContext(ctx).WithField("candleCount", len(candles)).WithField("err", err).Infof("Candles not found.")
		return nil, err
	}
	logger.FromContext(ctx).WithField("candleCount", len(candles)).Infof("Success get candles.")
	return candles, err
}

func (s Service) GetLastCandle(ctx context.Context, market string, interval string) (*domain.Chart, error) {
	cs, err := s.GetMinuteCandles(ctx, market)
	s.AggregateCandleToChartByInterval(cs, interval, 1)
	/*if err == nil {
		last := cs[len(cs)-1]
		return last, err
	}*/

	return nil, err
}

func (s Service) PushLastUpdatedCandle(ctx context.Context, market string, interval string) {
	upd, _ := s.GetLastCandle(ctx, market, interval)
	if upd != nil {
		s.UpdatedCandles <- upd
	}

}
func (s Service) AggregateCandleToChartByInterval(candles []*domain.Candle, interval string, count int) *domain.Chart {
	var chart *domain.Chart
	switch interval {
	case domain.Candle1MInterval:
		chart = s.aggregateMinCandlesToChart(candles, 1, count)
	case domain.Candle3MInterval:
		chart = s.aggregateMinCandlesToChart(candles, 3, count)
	case domain.Candle5MInterval:
		chart = s.aggregateMinCandlesToChart(candles, 5, count)
	case domain.Candle15MInterval:
		chart = s.aggregateMinCandlesToChart(candles, 15, count)
	case domain.Candle30MInterval:
		chart = s.aggregateMinCandlesToChart(candles, 30, count)
	case domain.Candle1HInterval:
		chart = s.aggregateHoursCandlesToChart(candles, 1, count)
	case domain.Candle2HInterval:
		chart = s.aggregateHoursCandlesToChart(candles, 2, count)
	case domain.Candle4HInterval:
		chart = s.aggregateHoursCandlesToChart(candles, 4, count)
	case domain.Candle6HInterval:
		chart = s.aggregateHoursCandlesToChart(candles, 6, count)
	case domain.Candle12HInterval:
		chart = s.aggregateHoursCandlesToChart(candles, 12, count)
	case domain.Candle1DInterval:
		chart = s.aggregateHoursCandlesToChart(candles, 24, count)
	case domain.Candle1MHInterval:
		chart = s.aggregateMonthCandlesToChart(candles, count)
	default :
		logger.FromContext(context.Background()).WithField("interval", interval).Errorf("Unsupported interval.")
	}

	return chart
}

func (s Service) aggregateMinCandlesToChart(candles []*domain.Candle, minute int, count int) *domain.Chart {
	result := make(map[int64]*domain.Candle)

	var min int
	var mod int
	var mul time.Duration
	var timestamp int64
	for _, candle := range candles {
		min = int(int64(candle.Timestamp.Minute()))
		mod = min % minute
		mul = time.Duration(mod) * -time.Minute
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


func (s Service) compare(c *domain.Candle, candle *domain.Candle) *domain.Candle {
	comparedCandle := &domain.Candle{}
	if c.Timestamp.Unix() < candle.Timestamp.Unix() {
		comparedCandle.Open = c.Open
		comparedCandle.Close = candle.Close
	} else {
		comparedCandle.Open = candle.Open
		comparedCandle.Close = c.Close
	}

	if c.High < candle.High {
		comparedCandle.High = candle.High
	}
	if c.Low > candle.Low {
		comparedCandle.Low = candle.Low
	}
	comparedCandle.Volume = c.Volume + candle.Volume
	comparedCandle.Timestamp = candle.Timestamp

	return comparedCandle
}

func (s *Service) aggregateHoursCandlesToChart(candles []*domain.Candle, hour int, count int) *domain.Chart {
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

func (s *Service) aggregateMonthCandlesToChart(candles []*domain.Candle, count int) *domain.Chart {
	result := make(map[int64]*domain.Candle)

	var timestamp int64
	for _, candle := range candles {
		timestamp = time.Date(candle.Timestamp.Year(), candle.Timestamp.Month(), 1, 0, 0, 0, 0, time.Local).Unix()
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
		O: make([]float64, 0),
		H: make([]float64, 0),
		L: make([]float64, 0),
		C: make([]float64, 0),
		V: make([]float64, 0),
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
