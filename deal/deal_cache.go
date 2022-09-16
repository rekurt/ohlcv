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

	return deal, nil
}

func (s *cacheService) GetLastTrades(ctx context.Context, symbol string, limit int32) ([]domain.Deal, error) {
	return s.under.GetLastTrades(ctx, symbol, limit)
}

func (s *cacheService) GetTickerPriceChangeStatistics(ctx context.Context, duration time.Duration, market string) ([]domain.TickerPriceChangeStatistics, error) {
	const op = "cacheService_GetTickerPriceChangeStatistics"

	result := make([]domain.TickerPriceChangeStatistics, 0)

	if market != "" {
		k := key{duration: duration, market: market}

		resp, ok := s.cache.Get(k)
		if !ok {
			logger.
				FromContext(ctx).
				WithField("op", op).
				WithField("market", market).
				Errorf("error getting 24hr ticker: no data in cache")

			return result, nil
		}

		if r, ok := resp.(domain.TickerPriceChangeStatistics); ok {
			return []domain.TickerPriceChangeStatistics{r}, nil
		}

		return result, nil
	}

	for _, m := range s.marketsMap {
		k := key{duration: duration, market: m}

		resp, ok := s.cache.Get(k)
		if !ok {
			logger.
				FromContext(ctx).
				WithField("op", op).
				WithField("market", m).
				Errorf("error getting 24hr ticker: no data in cache")

			continue
		}

		if r, ok := resp.(domain.TickerPriceChangeStatistics); ok {
			result = append(result, r)
		}
	}

	return result, nil
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
				WithField("svc", "cacheService").
				Errorf("Consuming session was finished with error", err)
		}
	}()
}

func (s *cacheService) LoadCache(ctx context.Context) {
	const op = "cacheService_LoadCache"

	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentTickers, err := s.under.GetTickerPriceChangeStatistics(ctx, 24*time.Hour, "")
			if err != nil {
				logger.FromContext(ctx).
					WithField("err", err).
					WithField("op", op).
					Errorf("loading cache with error", err)

				continue
			}

			for _, ct := range currentTickers {
				s.cache.Set(
					key{24 * time.Hour, ct.Symbol},
					ct,
					time.Second*30,
				)
			}
		}
	}
}

type key struct {
	duration time.Duration
	market   string
}
