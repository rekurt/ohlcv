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
	DealsDbCollection  *mongo.Collection
	CandleDbCollection *mongo.Collection
}

func (s Storage) GetCandles(
	ctx context.Context,
	market string,
	unit string,
	unitSize int,
	period ...time.Time,
) *domain.Chart {
	logger.FromContext(ctx).WithField(
		"market",
		market,
	).Infof("[CandleService] Call GetCandles")
	from, to := period[0], period[1]

	sortStage := bson.D{{"$sort", bson.D{
		{
			"t", -1,
		},
	}}}

	matchStage := bson.D{
		{"$match", bson.D{
			{"data.market", market},
			{"t", bson.D{
				{"$gte", primitive.NewDateTimeFromTime(from)},
				{"$lte", primitive.NewDateTimeFromTime(to)},
			}},
		}},
	}
	groupStage := bson.D{{"$group", bson.D{
		{"_id", bson.D{
			{"t", bson.D{
				{"$dateTrunc", bson.D{
					{"date", "$t"},
					{"unit", unit},
					{"binSize", unitSize},
				}},
			}},
			{"market", "$data.market"},
		}},
		{"h", bson.D{{"$max", "$data.price"}}},
		{"l", bson.D{{"$min", "$data.price"}}},
		{"o", bson.D{{"$first", "$data.price"}}},
		{"c", bson.D{{"$last", "$data.price"}}},
		{"v", bson.D{{"$sum", "$data.volume"}}},
		//{"t", bson.D{{"$first", "$t"}}},
	}}}

	projectStage := bson.D{
		{"$project", bson.D{
			{"_id", 0},
			{"t", "$_id.t"},
			{"market", "$_id.market"},
			{
				"ohlcv", bson.D{
					{"o", "$o"},
					{"h", "$h"},
					{"l", "$l"},
					{"c", "$c"},
					{"v", "$v"},
				},
			},

			/*{"t", bson.D{
				{"$dateTrunc", bson.D{
					{"date", "$t"},
					{"unit", unit},
					{"binSize", unitSize},
				}},
			}},*/
			//{"price", "$price"},
			//{"volume", "$volume"},
		}},
	}

	opts := options.Aggregate()
	adu := true
	opts.AllowDiskUse = &adu
	opts.Hint = "trades"
	//opts.SetMaxAwaitTime(30*time.Second)
	cursor, err := s.DealsDbCollection.Aggregate(
		ctx,
		mongo.Pipeline{sortStage, matchStage, groupStage, projectStage},
		opts,
	)
	if err != nil {
		logger.FromContext(ctx).WithField(
			"error",
			err,
		).Errorf("[CandleService]Failed apply a aggregation function on the collection.")
		return nil
	}

	test := make([]bson.M, 0)
	candles := make([]*domain.Candle, 0)

	err = cursor.All(ctx, &test)

	if err != nil {
		logger.FromContext(ctx).WithField(
			"error",
			err,
		).Errorf("[CandleService]Failed apply a aggregation function on the collection.")
		return nil
	}

	if len(candles) == 0 {
		logger.FromContext(ctx).WithField(
			"candleCount",
			len(candles),
		).WithField("err", err).WithField(
			"err",
			period,
		).Infof("Candles not found.")
		return nil
	}
	logger.FromContext(ctx).WithField(
		"candleCount",
		len(candles),
	).Infof("Success get candles.")


	return &domain.Chart{}
	//return candles
}
