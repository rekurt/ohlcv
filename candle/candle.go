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
}

func NewService(dealsDbCollection *mongo.Collection) *Service {
	return &Service{DealsDbCollection: dealsDbCollection}
}

func (s Service) Start(ctx context.Context)  {
	ticker := time.NewTicker(1 * time.Minute)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <- ticker.C:
				// do stuff
			case <- quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (s Service) GetMinuteCandles(ctx context.Context, market string) ([]domain.Candle, error) {
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
		{"timestamp", "$time"},
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
	}
	var candles []domain.Candle
	if err = cursor.All(ctx, &candles); err != nil {
		logger.FromContext(ctx).WithField("error", err).Errorf("[CandleService]Failed apply a aggregation function on the collection.")
	}

	return candles, err
}
