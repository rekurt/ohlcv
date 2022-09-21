package service

import (
	"bitbucket.org/novatechnologies/ohlcv/internal/model"
	"bitbucket.org/novatechnologies/ohlcv/internal/repository"
	"context"
	"time"
)

type Candle struct {
	candleRepository *repository.Candle
}

// NewCandle returns candle service
func NewCandle(candleRepository *repository.Candle) *Candle {
	return &Candle{candleRepository: candleRepository}
}

// GenerateMinuteCandles generates minute candles
func (s *Candle) GenerateMinuteCandles(ctx context.Context, from, to time.Time) ([]*model.Candle, error) {
	return s.candleRepository.GenerateMinuteCandles(ctx, from, to)

}
