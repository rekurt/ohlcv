package domain

import (
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewCurrentCandles(t *testing.T) {
	timeNow = func() time.Time {
		return time.Date(2020, 4, 14, 15, 45, 56, 0, time.UTC)
	}
	candles := NewCurrentCandles(context.Background())
	var candlesUpdated []CurrentCandle
	go func() {
		for candle := range candles.GetUpdates() {
			candlesUpdated = append(candlesUpdated, candle)
		}
	}()
	_, ok := candles.GetCandle("ETH/BTC", Candle1HResolution)
	assert.False(t, ok)
	require.NoError(t, candles.AddDeal(matcher.Deal{
		Market:    "ETH/BTC",
		CreatedAt: time.Date(2020, 4, 14, 15, 21, 2, 0, time.UTC).UnixMilli(),
		Price:     "0.015",
		Amount:    "134.5",
	}))
	require.NoError(t, candles.AddDeal(matcher.Deal{
		Market:    "ETH/BTC",
		CreatedAt: time.Date(2020, 4, 14, 15, 45, 50, 0, time.UTC).UnixMilli(),
		Price:     "0.019",
		Amount:    "14.9",
	}))
	//assert.Equal(t, "", candlesUpdated)
	candle, ok := candles.GetCandle("ETH/BTC", Candle1HResolution)
	assert.True(t, ok)
	assert.Equal(t, CurrentCandle{
		Symbol:    "ETH/BTC",
		Open:      0.015,
		High:      0.019,
		Low:       0.015,
		Close:     0.019,
		Volume:    134.5 + 14.9,
		Timestamp: time.Date(2020, 4, 14, 15, 0, 0, 0, time.UTC),
	}, candle)
	candle, ok = candles.GetCandle("ETH/BTC", Candle15MResolution)
	assert.True(t, ok)
	assert.Equal(t, CurrentCandle{
		Symbol:    "ETH/BTC",
		Open:      0.019,
		High:      0.019,
		Low:       0.019,
		Close:     0.019,
		Volume:    14.9,
		Timestamp: time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
	}, candle)
}
