package candle

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type Service struct {
	DealsDbCollection *mongo.Collection
	Markets           []string
	UpdatedCandles    chan *domain.Chart
}

func NewService(dealsDbCollection *mongo.Collection, markets []string) *Service {
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
		fmt.Printf("%s", upd.T[0])
		s.UpdatedCandles <- upd
	}

}

func (s Service) AggregateCandleToChartByInterval(candles []*domain.Candle, interval string, count int) *domain.Chart {
	var chart *domain.Chart
	switch interval {
	/*case domain.Candle1MInterval:
	result = candles*/
	case domain.Candle5MInterval:
		chart = s.aggregate5MinCandles(candles, count)
	}

	return chart
}

func (s Service) aggregate5MinCandles(candles []*domain.Candle, count int) *domain.Chart {
	result := make(map[int64]*domain.Candle)
	var min int
	var mod int
	var mul time.Duration
	var timestamp int64
	for _, candle := range candles {
		min = int(int64(candle.Timestamp.Minute()))
		mod = min % 5
		mul = time.Duration(mod) * -time.Minute
		timestamp = candle.Timestamp.Add(mul).Unix()
		println(timestamp)
		c := result[timestamp]
		if c != nil {
			if c.Timestamp.Unix() < candle.Timestamp.Unix() {
				result[timestamp].Open = c.Open
				result[timestamp].Close = candle.Close
			} else {
				result[timestamp].Open = candle.Open
				result[timestamp].Close = c.Close
			}

			if c.High < candle.High {
				result[timestamp].High = candle.High
			}
			if c.Low > candle.Low {
				result[timestamp].Low = candle.Low
			}
			result[timestamp].Volume = c.Volume + candle.Volume
			result[timestamp].Timestamp = candle.Timestamp
		} else {
			result[timestamp] = candle
		}
	}

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
