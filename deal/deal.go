package deal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	pubsub "bitbucket.org/novatechnologies/common/events"
	"bitbucket.org/novatechnologies/common/events/topics"
	"bitbucket.org/novatechnologies/common/infra/logger"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"bitbucket.org/novatechnologies/interfaces/matcher"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"bitbucket.org/novatechnologies/ohlcv/domain"
)

func TopicName(prefix string) string {
	return prefix + "_" + topics.MatcherMDDeals
}

type Service struct {
	DbCollection *mongo.Collection
	Markets      map[string]string
	eventManager domain.EventsBroker
}

func NewService(
	dbCollection *mongo.Collection,
	markets map[string]string,
	eventPublisher domain.EventsBroker,
) *Service {
	return &Service{
		DbCollection: dbCollection,
		Markets:      markets,
		eventManager: eventPublisher,
	}
}

func (s *Service) SaveDeal(
	ctx context.Context,
	dealMessage *matcher.Deal,
) (*domain.Deal, error) {
	defer func() {
		if r := recover(); r != "" {
			logger.FromContext(ctx).Errorf(r)
			// TODO: sending notification manually to the sentry or alternative.
		}
	}()

	if dealMessage.TakerOrderId == "" || dealMessage.MakerOrderId == "" {
		logger.FromContext(ctx).Infof("The deal have empty TakerOrderId or MakerOrderId field. Skip. Dont save to mongo.")
		return nil, nil
	}
	t := time.Unix(0, dealMessage.CreatedAt)
	marketName := s.Markets[dealMessage.Market]
	deal := &domain.Deal{
		T: primitive.NewDateTimeFromTime(t),
		Data: domain.DealData{
			Price:        domain.MustParseDecimal(dealMessage.Price),
			Volume:       domain.MustParseDecimal(dealMessage.Amount),
			DealId:       dealMessage.Id,
			Market:       marketName,
			IsBuyerMaker: dealMessage.IsBuyerMaker,
		},
	}
	if err := deal.Validate(); err != nil {
		return nil, err
	}

	_, err := s.DbCollection.InsertOne(ctx, deal)
	if err != nil {
		logger.FromContext(ctx).WithField(
			"error",
			err.Error(),
		).Errorf("[DealService]Failed save deal.", deal)
		return nil, err
	}
	var deals = make([]*domain.Deal, 1)
	deals[0] = deal
	go s.eventManager.Publish(domain.EvTypeDeals, domain.NewEvent(ctx, deals))

	return deal, nil
}

func (s *Service) GetLastTrades(
	ctx context.Context,
	symbol string,
	limit int32,
) ([]domain.Deal, error) {
	ctx, cancelFunc := context.WithTimeout(ctx, time.Second)
	defer cancelFunc()

	if strings.TrimSpace(symbol) == "" || limit <= 0 || limit >= 1000 {
		logger.FromContext(ctx).Infof(
			"Incorrect args: symbol='%s', limit=%d",
			symbol,
			limit,
		)
		return nil, nil
	}
	cursor, err := s.DbCollection.Find(
		ctx,
		bson.M{"data.market": symbol},
		options.Find().SetLimit(int64(limit)),
	)
	if err != nil {
		logger.FromContext(ctx).WithField(
			"error",
			err.Error(),
		).Errorf("[DealService]Failed GetLastTrades")
		return nil, err
	}
	var deals []domain.Deal
	err = cursor.All(ctx, &deals)
	if err != nil {
		logger.FromContext(ctx).WithField(
			"error",
			err.Error(),
		).Errorf("[DealService]Failed GetLastTrades")
		return nil, err
	}
	return deals, nil
}

func (s *Service) GetTickerPriceChangeStatistics(ctx context.Context, duration time.Duration, market string) (domain.TickerPriceChangeStatistics, error) {
	if strings.TrimSpace(market) == "" {
		return domain.TickerPriceChangeStatistics{}, fmt.Errorf("GetTickerPriceChangeStatistics: invalid market '%s'", market)
	}
	matchStage := bson.D{{Key: "$match", Value: bson.D{
		{"t", bson.D{
			{"$gte", primitive.NewDateTimeFromTime(time.Now().Add(-duration))},
		}},
		{
			"data.market",
			market,
		},
	}}}
	sortStage := bson.D{{"$sort", bson.D{
		{
			"t", 1,
		},
	}}}
	groupStage := bson.D{
		{"$group",
			bson.D{
				{"_id", nil},
				{"volume", bson.D{{"$sum", "$data.volume"}}},
				{"count", bson.D{{"$count", bson.M{}}}},
				{"highPrice", bson.D{{"$max", "$data.price"}}},
				{"lowPrice", bson.D{{"$min", "$data.price"}}},
				{"openPrice", bson.D{{"$first", "$data.price"}}},
				{"closePrice", bson.D{{"$last", "$data.price"}}},
				{"openTime", bson.D{{"$first", "$t"}}},
				{"closeTime", bson.D{{"$last", "$t"}}},
				{"firstId", bson.D{{"$first", "$data.dealid"}}},
				{"lastId", bson.D{{"$last", "$data.dealid"}}},
			},
		},
	}
	aggregate, err := s.DbCollection.Aggregate(
		ctx,
		mongo.Pipeline{matchStage, sortStage, groupStage},
		options.Aggregate().SetMaxTime(time.Second*4),
	)
	if err != nil {
		return domain.TickerPriceChangeStatistics{}, fmt.Errorf("GetTickerPriceChangeStatistics: Aggregate error '%w'", err)
	}
	var resp []bson.M
	if err = aggregate.All(ctx, &resp); err != nil {
		return domain.TickerPriceChangeStatistics{}, fmt.Errorf("GetTickerPriceChangeStatistics: aggregate.All error '%w'", err)
	}
	if len(resp) == 0 {
		return domain.TickerPriceChangeStatistics{}, nil
	}
	m := resp[0]
	return domain.TickerPriceChangeStatistics{
		Symbol:    market,
		LastPrice: m["closePrice"].(primitive.Decimal128).String(),
		OpenPrice: m["openPrice"].(primitive.Decimal128).String(),
		HighPrice: m["highPrice"].(primitive.Decimal128).String(),
		LowPrice:  m["lowPrice"].(primitive.Decimal128).String(),
		Volume:    m["volume"].(primitive.Decimal128).String(),
		OpenTime:  m["openTime"].(primitive.DateTime).Time().UnixMilli(),
		CloseTime: m["closeTime"].(primitive.DateTime).Time().UnixMilli(),
		FirstId:   m["firstId"].(string),
		LastId:    m["lastId"].(string),
		Count:     int(m["count"].(int32)),
	}, nil
}

func (s *Service) RunConsuming(
	ctx context.Context,
	consumer pubsub.Subscriber,
	topic string,
) {
	go func() {
		err := func() error {
			return consumer.Consume(
				ctx,
				topic,
				func(
					ctx context.Context,
					metadata map[string]string,
					msg []byte,
				) error {
					dealMessage := matcher.Deal{}
					if err := proto.Unmarshal(msg, &dealMessage); err != nil {
						logger.FromContext(ctx).
							WithField("method", "consumer.deals.Unmarshal").
							Errorf(err)

						return errors.Wrap(
							err,
							"unmarshal error with protobuf deals msg",
						)
					}

					if deal, err := s.SaveDeal(ctx, &dealMessage); err != nil {
						return errors.Wrapf(err, "while saving deal %v into DB", deal)
					} else {

						var deals = make([]*domain.Deal, 1)
						deals[0] = deal
						s.eventManager.Publish(
							domain.EvTypeDeals,
							domain.NewEvent(ctx, deals),
						)
					}
					return nil
				},
			)
		}()
		if err != nil {
			logger.FromContext(ctx).
				WithField("err", err).
				WithField("svc", "DealsService").
				Errorf("Consuming session was finished with error", err)
		}
	}()
}
