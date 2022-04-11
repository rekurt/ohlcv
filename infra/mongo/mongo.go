package mongo

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"reflect"
)

func NewMongoClient(ctx context.Context, config infra.MongoDbConfig) *mongo.Client {
	credential := options.Credential{
		Username: config.User,
		Password: config.Password,
	}
	uri := fmt.Sprintf("mongodb://%s:%s@%s", config.User, config.Password, config.Host)
	clientOptions := options.Client().ApplyURI(uri).
		SetAuth(credential).
		SetMaxPoolSize(100)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		logger.FromContext(ctx).Errorf("[infra.Mongo] Failed connect to mongo ERROR:", err)
		os.Exit(1)
	}

	return client
}

func InitDealCollection(ctx context.Context, client *mongo.Client, config infra.MongoDbConfig) {
	client.Database(config.DbName).Collection(config.DealCollectionName).Drop(ctx)
	metaField := "market"
	granularity := "minutes"
	opt := options.CreateCollection().SetTimeSeriesOptions(&options.TimeSeriesOptions{
		TimeField: "time",
		MetaField: &metaField,
		Granularity: &granularity,
	})

	err := client.Database(config.DbName).CreateCollection(ctx, config.DealCollectionName, opt)
	if err != nil {
		panic(err)
	}
	coll := client.Database(config.DbName).Collection(config.DealCollectionName)
	_, err = coll.Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys:    bson.D{{Key: "deal_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	)

}

func GetCollection(ctx context.Context, client *mongo.Client, config infra.MongoDbConfig) *mongo.Collection {
	logger.FromContext(ctx).Infof("[infra.Mongo] Try get collection", "CollectionName: ", config.DealCollectionName)
	col := client.Database(config.DbName).Collection(config.DealCollectionName)
	logger.FromContext(ctx).Infof("Collection type:", reflect.TypeOf(col))

	return col
}
