package candle

import (
	"context"
	"time"

	"bitbucket.org/novatechnologies/common/infra/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"bitbucket.org/novatechnologies/ohlcv/domain"
)

type Storage struct {
	DealsDbCollection *mongo.Collection
}

func (s Storage) GetMinuteCandles(
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

	projectStage := bson.D{
		{"$project", bson.D{
			{"time", bson.D{
				{"$dateTrunc", bson.D{
					{"date", "$time"},
					{"unit", "minute"},
					{"binSize", 1},
				}},
			}},
			{"price", "$price"},
			{"volume", "$volume"},
			{"market", "$market"},
		}},
	}

	groupStage := bson.D{{"$group", bson.D{
		{"_id", bson.D{
			{"market", "$market"},
			{"time", "$time"},
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

	opts := options.Aggregate()
	adu := true
	opts.AllowDiskUse = &adu
	cursor, err := s.DealsDbCollection.Aggregate(
		ctx,
		mongo.Pipeline{matchStage, projectStage, groupStage, sortStage},
		opts,
	)
	if err != nil {
		logger.FromContext(ctx).WithField(
			"error",
			err,
		).Errorf("[CandleService]Failed apply a aggregation function on the collection.")
		return nil, err
	}

	candles := make([]*domain.Candle, 0)

	err = cursor.All(ctx, &candles)

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
		).WithField("err", err).WithField(
			"err",
			period,
		).Infof("Candles not found.")
		return nil, err
	}
	logger.FromContext(ctx).WithField(
		"candleCount",
		len(candles),
	).Infof("Success get candles.")

	return candles, err
}
