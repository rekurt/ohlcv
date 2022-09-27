package repository

import (
	"bitbucket.org/novatechnologies/ohlcv/internal/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

// Candle repository working only with minutes collection
type Candle struct {
	dealsDBCollection *mongo.Collection
}

// NewCandle return new Candle repository
func NewCandle(dealsDBCollection *mongo.Collection) *Candle {
	return &Candle{dealsDBCollection: dealsDBCollection}
}
func (r *Candle) GenerateMinuteCandles(ctx context.Context, from, to time.Time) ([]*model.Candle, error) {
	matchStage := bson.D{
		{"$match", bson.D{
			{"t", bson.D{
				{"$gte", primitive.NewDateTimeFromTime(from)},
				{"$lte", primitive.NewDateTimeFromTime(to)},
			}},
		}},
	}
	firstSortStage := bson.D{{"$sort", bson.D{
		{
			"data.market", 1,
		},
		{
			"t", -1,
		},
	}}}
	firstGroupStage := bson.D{{"$group", bson.D{
		{"_id", bson.D{
			{"s", "$data.market"},
			{"t", bson.D{
				{"$dateTrunc", bson.D{
					{"date", "$t"},
					{"unit", model.MinuteUnit},
					{"binSize", 1},
				}},
			}},
		}},
		{"o", bson.D{{"$last", "$data.price"}}},
		{"h", bson.D{{"$max", "$data.price"}}},
		{"l", bson.D{{"$min", "$data.price"}}},
		{"c", bson.D{{"$first", "$data.price"}}},
		{"v", bson.D{{"$sum", "$data.volume"}}},
	}}}
	projectStage := bson.D{
		{"$project", bson.D{
			{"t", "$_id.t"},
			{"s", "$_id.s"},
			{"o", bson.D{{"$toDecimal", "$o"}}},
			{"h", bson.D{{"$toDecimal", "$h"}}},
			{"l", bson.D{{"$toDecimal", "$l"}}},
			{"c", bson.D{{"$toDecimal", "$c"}}},
			{"v", bson.D{{"$toDecimal", "$v"}}},
		}},
	}
	opts := options.Aggregate()
	adu := true
	opts.AllowDiskUse = &adu
	cursor, err := r.dealsDBCollection.Aggregate(
		ctx,
		mongo.Pipeline{matchStage, firstSortStage, firstGroupStage, projectStage},
		opts,
	)
	if err != nil {
		return nil, fmt.Errorf("can't generate minute candle %w", err)
	}
	data := make([]*model.Candle, 0)
	err = cursor.All(ctx, &data)
	if err != nil {
		return nil, fmt.Errorf("can't serialise candle %w", err)
	}
	return data, err
}
