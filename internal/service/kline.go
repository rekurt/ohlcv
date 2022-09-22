package service

import (
	"bitbucket.org/novatechnologies/ohlcv/internal/model"
	"bitbucket.org/novatechnologies/ohlcv/internal/repository"
	"context"
	"time"
)

type Kline struct {
	klineRps *repository.Kline
}

// NewKline instance of kline service
func NewKline(repository *repository.Kline) *Kline {
	return &Kline{klineRps: repository}
}

// Get the klines via specific parameters
func (s *Kline) Get(ctx context.Context, from, to time.Time) ([]*model.Kline, error) {
	return s.klineRps.Get(ctx, from, to)
}
