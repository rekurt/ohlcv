package main

import (
	"bitbucket.org/novatechnologies/common/events/topics"
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"bitbucket.org/novatechnologies/ohlcv/api/http"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/deal"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"bitbucket.org/novatechnologies/ohlcv/infra/mongo"
	"context"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
	"os"
	"os/signal"
	"strconv"
	"time"
)

func main() {
	ctx := infra.GetContext()
	ctx, _ = context.WithTimeout(ctx, time.Second*15)
	conf := infra.SetConfig(ctx, "./config/.env")

	group, ctx := errgroup.WithContext(ctx)

	consumer := infra.NewConsumer(ctx, conf.KafkaConfig)

	mongoDbClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)
	dealCollection := mongo.GetCollection(ctx, mongoDbClient, conf.MongoDbConfig)
	deal2Collection := mongo.GetCollection(ctx, mongoDbClient, conf.MongoDbConfig)
	dealService := deal.NewService(dealCollection)
	candleService := candle.NewService(deal2Collection)

	group.Go(func() error {
		//todo check topic name!
		return consumer.Consume(ctx, topics.MatcherMDOrders, func(ctx context.Context, msg []byte) error {
			orderDeal := matcher.Order{}
			if er := proto.Unmarshal(msg, &orderDeal); er != nil {
				logger.FromContext(ctx).WithField("method", "consumer.deals.Unmarshal").Errorf("%v", er)
				os.Exit(1)
			}

			floatAmount, _ := strconv.ParseFloat( orderDeal.Deal.Amount, 64)

			d := &domain.Deal{
				Price:  orderDeal.Deal.Price,
				Volume: floatAmount,
				DealId: orderDeal.Deal.Id,
				Market: orderDeal.Market,
				Time:   time.Unix(orderDeal.Deal.CreatedAt, 0),
			}

			dealService.SaveDeal(ctx, d)
			return nil
		})
	})

	candleService.Start(ctx)

	server := http.NewServer(candleService)
	server.Start(ctx)

	//shutdown
	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, os.Interrupt)

	_ = <-signalCh

	server.Stop(ctx)

	return
}
