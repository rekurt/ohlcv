package service

import (
	"bitbucket.org/novatechnologies/ohlcv/internal/model"
	"bitbucket.org/novatechnologies/ohlcv/internal/repository"
	"context"
	"fmt"
	"time"
)

const defaultLimit = 64

type Kline struct {
	klineRps *repository.Kline
}

// NewKline instance of kline service
func NewKline(repository *repository.Kline) *Kline {
	return &Kline{klineRps: repository}
}

// Get the klines via specific parameters
func (s *Kline) Get(ctx context.Context, symbol, interval string, fromTime, toTime *time.Time, limit int) ([]*model.Kline, error) {
	unit, unitSize := model.GetResolution(interval)
	duration := model.StrResolutionToDuration(interval)
	from, to := s.getDefaultTimeRange(duration, limit)
	if fromTime != nil {
		from = *fromTime
	}
	if toTime != nil {
		to = *toTime
	}
	if limit == 0 {
		limit = defaultLimit
	}
	klines, err := s.klineRps.Get(ctx, symbol, unit, from, to, limit, unitSize)
	if err != nil {
		return nil, fmt.Errorf("kline service %w", err)
	}
	return klines, nil
}

func (s *Kline) getDefaultTimeRange(candleDuration time.Duration, limit int) (time.Time, time.Time) {
	to := time.Now().Truncate(candleDuration)
	from := to.Add(-(candleDuration * time.Duration(limit)))
	return from, to
}
