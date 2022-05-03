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
	"time"
)

func NewMongoClient(
	ctx context.Context,
	config infra.MongoDbConfig,
) *mongo.Client {
	authDbName := config.AuthDbName
	if authDbName == "" {
		authDbName = config.DbName
	}
	host := config.Host
	username := config.User
	password := config.Password
	timeoutD := 60 * time.Second

	clientOptions := options.Client().
		SetMaxPoolSize(100).
		SetConnectTimeout(timeoutD)

	if username != "" && password != ""{
		credential := options.Credential{
			AuthSource: authDbName,
			Username:   username,
			Password:   password,
		}
		uri := fmt.Sprintf("mongodb://%s", config.Host)
		clientOptions.ApplyURI(uri).SetAuth(credential)

	} else {
		clientOptions.ApplyURI(fmt.Sprintf(
			"mongodb://%s",
			host,
		))
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
	collection := initCollection(ctx, client.Database(config.DbName), config.MinuteCandleCollectionName, getMinutesCollectionOptions())
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
