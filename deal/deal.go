package deal

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strings"
	"time"

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

func (s Service) SaveDeal(
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

func (s Service) GetLastTrades(
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
		bson.M{"market": symbol},
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

func (s Service) RunConsuming(
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
