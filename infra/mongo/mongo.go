package mongo

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"context"
	"fmt"
	"github.com/AlekSi/pointer"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
)

func NewMongoClient(
	ctx context.Context,
	config infra.MongoDbConfig,
) *mongo.Client {
	authDbName := config.AuthDbName
	if authDbName == "" {
		authDbName = config.DbName
	}
	username := config.User
	password := config.Password
	clientOptions := options.Client()

	if username != "" && password != "" {
		credential := options.Credential{
			AuthSource: authDbName,
			Username:   config.User,
			Password:   config.Password,
		}
		uri := fmt.Sprintf(
			"mongodb://%s:%s@%s",
			config.User,
			config.Password,
			config.Host,
		)

		clientOptions.ApplyURI(uri).
			SetAuth(credential).
			SetMaxPoolSize(100)
	} else {
		clientOptions.ApplyURI(fmt.Sprintf(
			"mongodb://%s",
			config.Host,
		)).SetMaxPoolSize(100)
	}

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

func InitDealsCollection(
	ctx context.Context,
	client *mongo.Client,
	config infra.MongoDbConfig,
) *mongo.Collection {
	collection := initCollection(ctx, client.Database(config.DbName), config.DealCollectionName, getDealsCollectionOptions())
	createIndex(ctx, collection,
		bson.D{
			{
				"data.market",
				1,
			},
			{
				"t",
				-1,
			}})
	return collection
}

func InitMinutesCollection(
	ctx context.Context,
	client *mongo.Client,
	config infra.MongoDbConfig,
) *mongo.Collection {
	collection := initCollection(ctx, client.Database(config.DbName), config.MinuteCandleCollectionName, getMinutesCollectionOptions())
	createIndex(ctx, collection,
		bson.D{
			{
				"symbol", 1,
			},
			{
				"t",
				-1,
			},
		})
	return collection
}

func createIndex(ctx context.Context, coll *mongo.Collection, keys bson.D) {
	_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: keys,
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
	if CollectionExist(ctx, client, config.DbName, cName) {
		return GetCollection(ctx, client, config, cName)
	}
	return InitDealsCollection(ctx, client, config)
}

func GetOrCreateMinutesCollection(ctx context.Context,
	client *mongo.Client,
	config infra.MongoDbConfig) *mongo.Collection {
	cName := config.MinuteCandleCollectionName
	if CollectionExist(ctx, client, config.DbName, cName) {
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
	return client.Database(config.DbName).Collection(collectionName)
}
