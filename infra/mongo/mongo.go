package mongo

import (
	"context"
	"fmt"
	"os"
	"time"

	"bitbucket.org/novatechnologies/common/infra/logger"
	"github.com/AlekSi/pointer"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"bitbucket.org/novatechnologies/ohlcv/infra"
)

func NewMongoClient(
	ctx context.Context,
	config infra.MongoDbConfig,
) *mongo.Client {
	credential := options.Credential{
		AuthSource: config.DbName,
		Username:   config.User,
		Password:   config.Password,
	}
	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s",
		config.User,
		config.Password,
		config.Host,
	)

	timeoutD := 60 * time.Second
	clientOptions := options.Client().ApplyURI(uri).
		SetAuth(credential).
		SetMaxPoolSize(100).
		SetConnectTimeout(timeoutD)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		logger.FromContext(ctx).Errorf(
			"[infra.Mongo] Failed connect to mongo ERROR:",
			err,
		)
		os.Exit(1)
	}

	return client
}

//InitDealCollection runs manually now
//goland:noinspection GoUnusedExportedFunction
func InitDealCollection(
	ctx context.Context,
	client *mongo.Client,
	config infra.MongoDbConfig,
) {
	_ = client.Database(config.DbName).Collection(config.DealCollectionName).Drop(ctx)
	opt := options.CreateCollection().SetTimeSeriesOptions(
		&options.TimeSeriesOptions{
			TimeField:   "time",
			MetaField:   pointer.ToString("market"),
			Granularity: pointer.ToString("minutes"),
		},
	)

	err := client.Database(config.DbName).CreateCollection(
		ctx,
		config.DealCollectionName,
		opt,
	)
	if err != nil {
		panic(err)
	}
	coll := client.Database(config.DbName).Collection(config.DealCollectionName)
	_, err = coll.Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys:    bson.D{{Key: "dealid", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	)

}


func InitMinuteCandleCollection(
	ctx context.Context,
	client *mongo.Client,
	config infra.MongoDbConfig,
) {
	_ = client.Database(config.DbName).Collection(config.DealCollectionName).Drop(ctx)
	opt := options.CreateCollection().SetTimeSeriesOptions(
		&options.TimeSeriesOptions{
			TimeField:   "time",
			MetaField:   pointer.ToString("market"),
			Granularity: pointer.ToString("minutes"),
		},
	)

	err := client.Database(config.DbName).CreateCollection(
		ctx,
		config.MinuteCandleCollectionName,
		opt,
	)
	if err != nil {
		panic(err)
	}
	_ = client.Database(config.DbName).Collection(config.MinuteCandleCollectionName)
}

func GetCollection(
	ctx context.Context,
	client *mongo.Client,
	config infra.MongoDbConfig,
	collectionName string,
) *mongo.Collection {
	logger.FromContext(ctx).Infof(
		"[infra.Mongo] Try get collection %s",
		collectionName,
	)
	return client.Database(config.DbName).Collection(collectionName)
}
