package deal

import (
	"context"
	"time"

	pubsub "bitbucket.org/novatechnologies/common/events"
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"github.com/akyoto/cache"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

type cacheService struct {
	under      domain.Service
	cache      *cache.Cache
	marketsMap map[string]string
}

func NewCacheService(
	under domain.Service,
	marketsMap map[string]string,
) *cacheService {
	return &cacheService{
		under:      under,
		cache:      cache.New(time.Second * 10),
		marketsMap: marketsMap,
	}
}

func (s *cacheService) SaveDeal(ctx context.Context, dealMessage *matcher.Deal) (*domain.Deal, error) {
	deal, err := s.under.SaveDeal(ctx, dealMessage)
	if err != nil {
		return nil, err
	}

	symbol := s.marketsMap[dealMessage.Market]

	currentTicker, err := s.under.GetTickerPriceChangeStatistics(ctx, 24*time.Hour, symbol)
	if err != nil {
		return nil, err
	}

	s.cache.Set(
		key{24 * time.Hour, symbol},
		currentTicker,
		time.Second*30,
	)

	return deal, nil
}

func (s *cacheService) GetLastTrades(ctx context.Context, symbol string, limit int32) ([]domain.Deal, error) {
	return s.under.GetLastTrades(ctx, symbol, limit)
}

func (s *cacheService) GetTickerPriceChangeStatistics(ctx context.Context, duration time.Duration, market string) ([]domain.TickerPriceChangeStatistics, error) {
	if market != "" {
		return s.getTickerPriceChangeStatisticsByMarket(ctx, duration, market)
	}

	result := make([]domain.TickerPriceChangeStatistics, 0, len(s.marketsMap))

	for _, m := range s.marketsMap {
		resp, err := s.getTickerPriceChangeStatisticsByMarket(ctx, duration, m)
		if err != nil {
			logger.
				FromContext(ctx).
				WithField("op", "GetTickerPriceChangeStatistics").
				Errorf("error getting 24hr ticker: %s", err)

			continue
		}

		result = append(result, resp...)
	}

	return result, nil
}

func (s *cacheService) getTickerPriceChangeStatisticsByMarket(ctx context.Context, duration time.Duration, market string) ([]domain.TickerPriceChangeStatistics, error) {
	k := key{duration: duration, market: market}

	resp, ok := s.cache.Get(k)
	if !ok {
		underResp, err := s.under.GetTickerPriceChangeStatistics(ctx, duration, market)
		if err != nil {
			return nil, err
		}
		s.cache.Set(k, underResp, time.Second*30)
		return underResp, nil
	}

	return resp.([]domain.TickerPriceChangeStatistics), nil
}

func (s *cacheService) GetAvgPrice(ctx context.Context, duration time.Duration, market string) (string, error) {
	return s.under.GetAvgPrice(ctx, duration, market)
}

func (s *cacheService) RunConsuming(ctx context.Context, consumer pubsub.Subscriber, topic string, currentCandles candle.CurrentCandles) {
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
					err := currentCandles.AddDeal(dealMessage)
					if err != nil {
						logger.FromContext(ctx).
							WithField("method", "currentCandles.AddDeal in consuming").
							Errorf(err)
					}
					if deal, err := s.SaveDeal(ctx, &dealMessage); err != nil {
						return errors.Wrapf(err, "while saving deal %v into DB", deal)
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

type key struct {
	duration time.Duration
	market   string
}
