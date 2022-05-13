package domain

import (
	"context"
	"time"

	"bitbucket.org/novatechnologies/interfaces/matcher"
)

type Service interface {
	SaveDeal(ctx context.Context, dealMessage *matcher.Deal) (*Deal, error)
	GetLastTrades(ctx context.Context, symbol string, limit int32) ([]Deal, error)
	GetTickerPriceChangeStatistics(ctx context.Context, duration time.Duration, market string) ([]TickerPriceChangeStatistics, error)
}
