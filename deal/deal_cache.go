package deal

import (
	"context"
	"time"

	"bitbucket.org/novatechnologies/interfaces/matcher"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"github.com/akyoto/cache"
)

type cacheService struct {
	under domain.Service
	cache *cache.Cache
}

func NewCacheService(under domain.Service) *cacheService {
	return &cacheService{under: under, cache: cache.New(time.Second * 10)}
}

func (s *cacheService) SaveDeal(ctx context.Context, dealMessage *matcher.Deal) (*domain.Deal, error) {
	return s.under.SaveDeal(ctx, dealMessage)
}

func (s *cacheService) GetLastTrades(ctx context.Context, symbol string, limit int32) ([]domain.Deal, error) {
	return s.under.GetLastTrades(ctx, symbol, limit)
}

func (s *cacheService) GetTickerPriceChangeStatistics(ctx context.Context, duration time.Duration, market string) ([]domain.TickerPriceChangeStatistics, error) {
	type key struct {
		duration time.Duration
		market   string
	}
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
