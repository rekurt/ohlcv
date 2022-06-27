package mongo

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"context"
	"github.com/AlekSi/pointer"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

func NewMongoClient(
	ctx context.Context,
	config infra.MongoDbConfig,
) *mongo.Client {
	timeoutD := 60 * time.Second
	serverAPIOptions := options.ServerAPI(options.ServerAPIVersion1)

	clientOptions := options.Client().
		ApplyURI(config.ConnectionUrl).
		SetServerAPIOptions(serverAPIOptions).
		SetMaxPoolSize(100).
		SetConnectTimeout(timeoutD)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)

	if err != nil {
		logger.FromContext(ctx).Errorf("[infra.Mongo] Failed connect to mongo ERROR:",
			err,
		)
	}

	return client
}

func InitDealsCollection(
	ctx context.Context,
	client *mongo.Client,
	config infra.MongoDbConfig,
) *mongo.Collection {
	collection := initCollection(ctx, client.Database(config.DatabaseName), config.DealCollectionName, getDealsCollectionOptions())
	createIndex(ctx, collection, "trades",
		bson.D{
			{
				"data.market",
				1,
			},
			{
				"t",
				-1,
			}}, false)

	// Unique indexes are not supported on collections clustered by _id ;c
	// createIndex(ctx, collection, "dealid",
	//	bson.D{
	//		{
	//			"data.dealid",
	//			1,
	//		}}, true)

	return collection
}

func InitMinutesCollection(
	ctx context.Context,
	client *mongo.Client,
	config infra.MongoDbConfig,
) *mongo.Collection {
	collection := initCollection(ctx, client.Database(config.DatabaseName), config.MinuteCandleCollectionName, getMinutesCollectionOptions())
	createIndex(ctx, collection, "minutes",
		bson.D{
			{
				"symbol", 1,
			},
			{
				"t",
				-1,
			},
		}, false)
	return collection
}

func createIndex(ctx context.Context, coll *mongo.Collection, name string, keys bson.D, isUnique bool) {
	_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    keys,
		Options: options.Index().SetName(name).SetUnique(isUnique),
	})
	if err != nil {
		panic(err)
	}
}

func initCollection(ctx context.Context, db *mongo.Database, collectionName string, opt *options.CreateCollectionOptions) *mongo.Collection {
	err := db.CreateCollection(
		ctx,
		collectionName,
		opt,
	)
	if err != nil {
		panic(err)
	}

	return db.Collection(collectionName)
}

func getDealsCollectionOptions() *options.CreateCollectionOptions {
	return options.CreateCollection().SetTimeSeriesOptions(
		&options.TimeSeriesOptions{
			TimeField:   "t",
			MetaField:   pointer.ToString("data"),
			Granularity: pointer.ToString("hours"),
		},
	)
}

func getMinutesCollectionOptions() *options.CreateCollectionOptions {
	return options.CreateCollection()
}

func CollectionExist(ctx context.Context, client *mongo.Client, dbName string, collectionName string) bool {
	db := client.Database(dbName)
	collections, err := db.ListCollections(ctx, bson.M{})
	if err != nil {
		panic(err)
	}

	for collections.Next(ctx) {
		var collectionInfo struct {
			Name string `bson:"name"`
		}
		err := collections.Decode(&collectionInfo)
		if err != nil {
			panic(err)
		}

		if collectionInfo.Name == collectionName {
			return true
		}
	}

	return false
}

func GetOrCreateDealsCollection(ctx context.Context,
	client *mongo.Client,
	config infra.MongoDbConfig) *mongo.Collection {
	cName := config.DealCollectionName
	if CollectionExist(ctx, client, config.DatabaseName, cName) {
		return GetCollection(ctx, client, config, cName)
	}
	return InitDealsCollection(ctx, client, config)
}

func GetOrCreateMinutesCollection(ctx context.Context,
	client *mongo.Client,
	config infra.MongoDbConfig) *mongo.Collection {
	cName := config.MinuteCandleCollectionName
	if CollectionExist(ctx, client, config.DatabaseName, cName) {
		return GetCollection(ctx, client, config, cName)
	}
	return InitMinutesCollection(ctx, client, config)
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
	return client.Database(config.DatabaseName).Collection(collectionName)
}
