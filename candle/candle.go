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
	UpdatedCandles chan domain.Candle

}

func NewService(dealsDbCollection *mongo.Collection) *Service {
	return &Service{DealsDbCollection: dealsDbCollection}
}

func (s Service) Start(ctx context.Context)  {
	ticker := time.NewTicker(10 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <- ticker.C:
				logger.FromContext(ctx).WithField("market", "BTC-USDT").Infof("Get new candle.")
				candle, _ := s.GetLastCandle(ctx, "BTC-USDT")
				s.UpdatedCandles <- candle
			case <- quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (s Service) GetMinuteCandles(ctx context.Context, market string) ([]domain.Candle, error) {
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
	}

	var candles []domain.Candle
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

func (s Service) GetLastCandle(ctx context.Context, market string) (domain.Candle, error) {
	cs, err := s.GetMinuteCandles(ctx, market)
	last := cs[len(cs)-1]

	return last, err

}

func (s Service) Observe(ctx context.Context) domain.Candle {
	return <- s.UpdatedCandles
}
