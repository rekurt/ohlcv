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

	firstSortStage := bson.D{{"$sort", bson.D{
		{
			"data.market", 1,
		},
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

	firstGroupStage := bson.D{{"$group", bson.D{
		{"_id", bson.D{
			{"symbol", "$data.market"},
			{"t", bson.D{
				{"$dateTrunc", bson.D{
					{"date", "$t"},
					{"unit", unit},
					{"binSize", unitSize},
				}},
			}},
		}},
		{"h", bson.D{{"$max", "$data.price"}}},
		{"l", bson.D{{"$min", "$data.price"}}},
		{"o", bson.D{{"$first", "$data.price"}}},
		{"c", bson.D{{"$last", "$data.price"}}},
		{"v", bson.D{{"$sum", "$data.volume"}}},
	}}}
	tInt := bson.D{{"$toLong", "$_id.t"}}
	projectStage := bson.D{
		{"$project", bson.D{
			{"_id", 0},
			{"t", bson.D{{"$divide", []interface{}{tInt, 1000}}}},
			{"symbol", "$_id.symbol"},
			{"o", bson.D{{"$toDecimal", "$o"}}},
			{"h", bson.D{{"$toDecimal", "$h"}}},
			{"l", bson.D{{"$toDecimal", "$l"}}},
			{"c", bson.D{{"$toDecimal", "$c"}}},
			{"v", bson.D{{"$toDecimal", "$v"}}},
		}},
	}
	secondSortStage := bson.D{{"$sort", bson.D{
		{
			"symbol", 1,
		},
		{
			"t", 1,
		},
	}}}

	secondGroupStage := bson.D{{"$group", bson.D{
		{"_id", "$symbol"},
		{"h", bson.D{{"$push", bson.D{{"$toString", "$h"}}}}},
		{"l", bson.D{{"$push", "$l"}}},
		{"o", bson.D{{"$push", "$o"}}},
		{"c", bson.D{{"$push", "$c"}}},
		{"v", bson.D{{"$push", "$v"}}},
		{"t", bson.D{{"$push", "$t"}}},
	}}}

	opts := options.Aggregate()
	adu := true
	opts.AllowDiskUse = &adu
	opts.Hint = "trades"
	//opts.SetMaxAwaitTime(30*time.Second)
	cursor, err := s.DealsDbCollection.Aggregate(
		ctx,
		mongo.Pipeline{matchStage, firstSortStage, firstGroupStage, projectStage, secondSortStage, secondGroupStage},
		opts,
	)

	if err != nil {
		logger.FromContext(ctx).WithField(
			"error",
			err,
		).Errorf("[CandleService]Failed apply a aggregation function on the collection.", err)
		return nil
	}

	data := make([]*domain.Chart, 0)
	err = cursor.All(ctx, &data)

	if err != nil {
		logger.FromContext(ctx).WithField(
			"error",
			err,
		).Errorf("[CandleService]Failed apply a aggregation function on the collection.", err)
		return nil
	}

	if len(data) == 0 {
		logger.FromContext(ctx).WithField(
			"candleCount",
			0,
		).WithField("err", err).WithField(
			"err",
			period,
		).Infof("Candles not found.")
		return nil
	}
	chart := data[0]
	logger.FromContext(ctx).WithField(
		"candleCount",
		len(chart.T),
	).Infof("Success get candles.")
	chart.SetMarket(market)
	return chart
}
