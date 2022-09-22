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

type Kline struct {
	dealsDbCollection *mongo.Collection
}

// NewKline creates kline repository
func NewKline(dealsDbCollection *mongo.Collection) *Kline {
	return &Kline{dealsDbCollection: dealsDbCollection}
}

// Get klines according parameters
func (r *Kline) Get(ctx context.Context, from, to time.Time) ([]*model.Kline, error) {
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
			{"symbol", "$data.market"},
			{"openTime", bson.D{
				{"$dateTrunc", bson.D{
					{"date", "$t"},
					{"unit", model.MinuteUnit},
					{"binSize", 1},
				}},
			}},
		}},
		{"open", bson.D{{"$last", "$data.price"}}},
		{"high", bson.D{{"$max", "$data.price"}}},
		{"low", bson.D{{"$min", "$data.price"}}},
		{"close", bson.D{{"$first", "$data.price"}}},
		{"volume", bson.D{{"$sum", "$data.volume"}}},
		{"quote", bson.D{{"$sum", bson.D{{"$multiply", bson.A{"$data.price", "$data.volume"}}}}}},
		{"trades", bson.D{{"$count", bson.D{}}}},
		{"takerAssets", bson.D{{"$sum", bson.D{
			{"$switch", bson.D{
				{"branches", bson.A{
					bson.D{
						{"case", bson.D{
							{"$eq", bson.A{"$data.isbuyermaker", true}}}},
						{"then", "$data.volume"},
					}}},
				{"default", 0}},
			},
		}}}},
		{"takerQuotes", bson.D{{"$sum", bson.D{
			{"$switch", bson.D{
				{"branches", bson.A{
					bson.D{
						{"case", bson.D{
							{"$eq", bson.A{"$data.isbuyermaker", true}}}},
						{"then", bson.D{{"$multiply", bson.A{"$data.price", "$data.volume"}}}},
					}}},
				{"default", 0}},
			},
		}}}},
	}}}

	projectStage := bson.D{
		{"$project", bson.D{
			{"openTime", "$_id.openTime"},
			{"closeTime", bson.D{{
				"$dateAdd", bson.D{
					{"startDate", "$_id.openTime"},
					{"unit", model.MinuteUnit},
					{"amount", 1},
				},
			}}},
			{"symbol", "$_id.symbol"},
			{"open", "$open"},
			{"high", "$high"},
			{"low", "$low"},
			{"close", "$close"},
			{"volume", "$volume"},
			{"quote", "$quote"},
			{"trades", "$trades"},
			{"takerAssets", "$takerAssets"},
			{"takerQuotes", "$takerQuotes"},
		}},
	}

	opts := options.Aggregate()
	adu := true
	opts.AllowDiskUse = &adu
	opts.Hint = "trades"
	cursor, err := r.dealsDbCollection.Aggregate(
		ctx,
		mongo.Pipeline{matchStage, firstSortStage, firstGroupStage, projectStage},
		opts,
	)

	if err != nil {
		return nil, fmt.Errorf("failed apply a kline aggregation function on the collection. %w", err)
	}

	data := make([]*model.Kline, 0)
	err = cursor.All(ctx, &data)

	if err != nil {
		return nil, fmt.Errorf("failed parse kline result to collection. %w", err)
	}
	return data, nil
}
