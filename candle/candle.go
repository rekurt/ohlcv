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
	Markets           []string
	UpdatedCandles    chan *domain.Candle
}

func NewService(dealsDbCollection *mongo.Collection, markets []string) *Service {
	return &Service{DealsDbCollection: dealsDbCollection, Markets: markets}
}

//Generation for websocket pushing new candle every min (example: empty candles)
func (s Service) CronCandleGenerationStart(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Second)
		done := make(chan bool)
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				for _, market := range s.Markets {
					logger.FromContext(ctx).WithField("market", market).Infof("[CronCandleGenerationStart]Getting new candle for the market.")
					s.PushLastUpdatedCandle(ctx, market)
				}
			default:
				logger.FromContext(ctx).Infof("[CronCandleGenerationStart]Waiting...")
				time.Sleep(10 *time.Second)
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

func (s Service) GetLastCandle(ctx context.Context, market string) (*domain.Candle, error) {
	cs, err := s.GetMinuteCandles(ctx, market)
	if err == nil {
		last := cs[len(cs)-1]
		return last, err
	}

	return nil, err
}

func (s Service) PushLastUpdatedCandle(ctx context.Context, market string) {
	candle, _ := s.GetLastCandle(ctx, market)
	if candle != nil {
		s.UpdatedCandles <- candle
	}
}
