package domain

import (
	"context"

	"bitbucket.org/novatechnologies/interfaces/matcher"
)

type Service interface {
	SaveDeal(ctx context.Context, dealMessage matcher.Deal) (*Deal, error)
	GetLastTrades(ctx context.Context, symbol string, limit int32) ([]Deal, error)
}
