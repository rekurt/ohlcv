package service

import (
	"context"
	"time"

	"bitbucket.org/novatechnologies/ohlcv/internal/model"
	"bitbucket.org/novatechnologies/ohlcv/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"

	pubsub "bitbucket.org/novatechnologies/common/events"
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

type Deal struct {
	under       *repository.Deal
	tickerCache *repository.Ticker
	marketsMap  map[string]string
	eventChanel chan *model.Deal
}

func NewDeal(
	dealRepository *repository.Deal,
	tickerCache *repository.Ticker,
	marketsMap map[string]string,
	eventChanel chan *model.Deal,
) *Deal {
	return &Deal{
		under:       dealRepository,
		tickerCache: tickerCache,
		marketsMap:  marketsMap,
		eventChanel: eventChanel,
	}
}

func (s *Deal) SaveDeal(ctx context.Context, dealMessage *matcher.Deal) (*model.Deal, error) {
	if dealMessage.TakerOrderId == "" || dealMessage.MakerOrderId == "" {
		logger.FromContext(ctx).Infof("The deal have empty TakerOrderId or MakerOrderId field. Skip. Dont save to mongo.")
		return nil, nil
	}
	t := time.Unix(0, dealMessage.CreatedAt)
	marketName := s.marketsMap[dealMessage.Market]
	deal := &model.Deal{
		T: primitive.NewDateTimeFromTime(t),
		Data: model.DealData{
			Price:        model.MustParseDecimal(dealMessage.Price),
			Volume:       model.MustParseDecimal(dealMessage.Amount),
			DealId:       dealMessage.Id,
			Market:       marketName,
			IsBuyerMaker: dealMessage.IsBuyerMaker,
		},
	}
	if err := deal.Validate(); err != nil {
		return nil, err
	}

	err := s.under.Save(ctx, deal)
	if err != nil {
		return nil, err
	}
	select {
	case s.eventChanel <- deal:
	default:
		logger.FromContext(ctx).Errorf("deal channel overloaded")
	}
	return deal, nil
}

func (s *Deal) GetTickerPriceChangeStatistics(ctx context.Context, market string) ([]*domain.TickerPriceChangeStatistics, error) {
	const op = "cacheService_GetTickerPriceChangeStatistics"

	result := make([]*domain.TickerPriceChangeStatistics, 0)

	if market != "" {
		resp, ok := s.tickerCache.Get(market)
		if !ok {
			logger.
				FromContext(ctx).
				WithField("op", op).
				WithField("market", market).
				Errorf("error getting 24hr ticker: no data in cache")

			return result, nil
		}

		return []*domain.TickerPriceChangeStatistics{resp}, nil
	}

	for _, m := range s.marketsMap {
		resp, ok := s.tickerCache.Get(m)
		if !ok {
			logger.
				FromContext(ctx).
				WithField("op", op).
				WithField("market", m).
				Errorf("error getting 24hr ticker: no data in cache")

			continue
		}

		result = append(result, resp)
	}

	return result, nil
}

func (s *Deal) GetAvgPrice(ctx context.Context, duration time.Duration, market string) (string, error) {
	return s.under.GetAvgPrice(ctx, duration, market)
}

func (s *Deal) GetLastTrades(ctx context.Context, symbol string, limit int32) ([]*model.Deal, error) {
	return s.under.GetLastTrades(ctx, symbol, limit)
}

func (s *Deal) RunConsuming(ctx context.Context, consumer pubsub.Subscriber, topic string, currentCandles candle.CurrentCandles) {
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
					dealMessage := &matcher.Deal{}
					if err := proto.Unmarshal(msg, dealMessage); err != nil {
						logger.FromContext(ctx).
							WithField("method", "consumer.deals.Unmarshal").
							Errorf(err)

						return errors.Wrap(
							err,
							"unmarshal error with protobuf deals msg",
						)
					}
					err := currentCandles.AddDeal(dealMessage)
					if err != nil {
						logger.FromContext(ctx).
							WithField("method", "currentCandles.AddDeal in consuming").
							Errorf(err)
					}
					if deal, err := s.SaveDeal(ctx, dealMessage); err != nil {
						return errors.Wrapf(err, "while saving deal %v into DB", deal)
					}
					return nil
				},
			)
		}()
		if err != nil {
			logger.FromContext(ctx).
				WithField("err", err).
				WithField("svc", "Deal").
				Errorf("Consuming session was finished with error", err)
		}
	}()
}

func (s *Deal) LoadCache(ctx context.Context) {
	const op = "cacheService_LoadCache"
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentTickers, err := s.under.GetTickerPriceChangeStatistics(ctx, "")
			if err != nil {
				logger.FromContext(ctx).
					WithField("err", err).
					WithField("op", op).
					Errorf("loading cache with error", err)

				continue
			}
			for _, ct := range currentTickers {
				s.tickerCache.Set(ct.Symbol, ct)
			}
		}
	}
}
