package candle

import (
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewCurrentCandles_GetCandle(t *testing.T) {
	timeNow = func() time.Time {
		return time.Date(2020, 4, 14, 15, 45, 56, 0, time.UTC)
	}
	candles := NewCurrentCandles(context.Background())
	//make 2 deals in ETH/BTC
	require.NoError(t, candles.AddDeal(matcher.Deal{
		Market:    "ETH/BTC",
		CreatedAt: time.Date(2020, 4, 14, 15, 21, 2, 0, time.UTC).UnixNano(),
		Price:     "0.015",
		Amount:    "134.5",
	}))
	require.NoError(t, candles.AddDeal(matcher.Deal{
		Market:    "ETH/BTC",
		CreatedAt: time.Date(2020, 4, 14, 15, 45, 50, 0, time.UTC).UnixNano(),
		Price:     "0.019",
		Amount:    "14.9",
	}))
	assert.Equal(t,
		CurrentCandle{
			Symbol:    "ETH/BTC",
			Open:      0.015,
			High:      0.019,
			Low:       0.015,
			Close:     0, //0 because not closed yet
			Volume:    134.5 + 14.9,
			OpenTime:  time.Date(2020, 4, 14, 15, 0, 0, 0, time.UTC),
			CloseTime: time.Date(2020, 4, 14, 16, 0, 0, 0, time.UTC),
		},
		candles.GetCandle("ETH/BTC", domain.Candle1HResolution),
	)
	assert.Equal(t,
		CurrentCandle{
			Symbol:    "ETH/BTC",
			Open:      0.019,
			High:      0.019,
			Low:       0.019,
			Close:     0,
			Volume:    14.9,
			OpenTime:  time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
			CloseTime: time.Date(2020, 4, 14, 16, 0, 0, 0, time.UTC),
		},
		candles.GetCandle("ETH/BTC", domain.Candle15MResolution),
	)
	assert.Equal(t,
		CurrentCandle{
			Symbol:    "ETH/RUB",
			OpenTime:  time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
			CloseTime: time.Date(2020, 4, 14, 16, 0, 0, 0, time.UTC),
		},
		candles.GetCandle("ETH/RUB", domain.Candle15MResolution),
		"no trades in this market",
	)
	//time goes on, but no trades
	timeNow = func() time.Time {
		return time.Date(2020, 4, 14, 16, 23, 56, 0, time.UTC)
	}
	assert.Equal(t,
		CurrentCandle{
			Symbol:    "ETH/BTC",
			OpenTime:  time.Date(2020, 4, 14, 16, 23, 0, 0, time.UTC),
			CloseTime: time.Date(2020, 4, 14, 16, 24, 0, 0, time.UTC),
		},
		candles.GetCandle("ETH/BTC", domain.Candle1MResolution),
		"return empty candle if no trades",
	)
	assert.Equal(t,
		CurrentCandle{
			Symbol:    "ETH/BTC",
			OpenTime:  time.Date(2020, 4, 14, 16, 0, 0, 0, time.UTC),
			CloseTime: time.Date(2020, 4, 14, 20, 0, 0, 0, time.UTC),
		},
		candles.GetCandle("ETH/BTC", domain.Candle4H2Resolution),
		"return empty candle if no trades",
	)
}

func TestNewCurrentCandles_refreshAll(t *testing.T) {
	timeNow = func() time.Time {
		return time.Date(2020, 4, 14, 15, 45, 56, 0, time.UTC)
	}
	candles := NewCurrentCandles(context.Background()).(*currentCandles)
	require.NoError(t, candles.AddDeal(matcher.Deal{
		Market:    "ETH/BTC",
		CreatedAt: time.Date(2020, 4, 14, 15, 45, 50, 0, time.UTC).UnixNano(),
		Price:     "0.019",
		Amount:    "14.9",
	}))
	assert.Equal(t,
		CurrentCandle{
			Symbol:    "ETH/BTC",
			Open:      0.019,
			High:      0.019,
			Low:       0.019,
			Close:     0,
			Volume:    14.9,
			OpenTime:  time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
			CloseTime: time.Date(2020, 4, 14, 16, 0, 0, 0, time.UTC),
		},
		candles.GetCandle("ETH/BTC", domain.Candle15MResolution),
	)
	candles.refreshAll()
	//filled candle is still current
	assert.Equal(t,
		CurrentCandle{
			Symbol:    "ETH/BTC",
			Open:      0.019,
			High:      0.019,
			Low:       0.019,
			Close:     0,
			Volume:    14.9,
			OpenTime:  time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
			CloseTime: time.Date(2020, 4, 14, 16, 0, 0, 0, time.UTC),
		},
		candles.GetCandle("ETH/BTC", domain.Candle15MResolution),
	)
	//change time
	timeNow = func() time.Time {
		return time.Date(2020, 4, 14, 17, 45, 56, 0, time.UTC)
	}
	candles.refreshAll()
	//no trades long ago, only empty candle
	assert.Equal(t,
		CurrentCandle{
			Symbol:    "ETH/RUB",
			OpenTime:  time.Date(2020, 4, 14, 17, 45, 0, 0, time.UTC),
			CloseTime: time.Date(2020, 4, 14, 18, 0, 0, 0, time.UTC),
		},
		candles.GetCandle("ETH/RUB", domain.Candle15MResolution),
	)
}

func TestNewCurrentCandles_updates(t *testing.T) {
	getAvailableResolutions = func() []string {
		return []string{domain.Candle1MResolution, domain.Candle1HResolution}
	}
	now := time.Date(2020, 4, 14, 15, 45, 56, 0, time.UTC)
	timeNow = func() time.Time {
		return now
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	candles := NewCurrentCandles(ctx).(*currentCandles)
	updates := candles.GetUpdates()
	require.NoError(t, candles.AddDeal(matcher.Deal{
		Market:    "ETH/BTC",
		CreatedAt: time.Date(2020, 4, 14, 15, 45, 50, 0, time.UTC).UnixNano(),
		Price:     "0.019",
		Amount:    "14.9",
	}))
	require.Len(t, updates, len(getAvailableResolutions()))
	candle, ok := <-updates
	assert.True(t, ok)
	assert.Equal(t,
		CurrentCandle{
			Symbol:    "ETH/BTC",
			Open:      0.019,
			High:      0.019,
			Low:       0.019,
			Close:     0.019,
			Volume:    14.9,
			OpenTime:  time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
			CloseTime: time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
		}, candle)

	candle, ok = <-updates
	assert.True(t, ok)
	assert.Equal(t,
		CurrentCandle{
			Symbol:    "ETH/BTC",
			Open:      0.019,
			High:      0.019,
			Low:       0.019,
			Close:     0.019,
			Volume:    14.9,
			OpenTime:  time.Date(2020, 4, 14, 15, 0, 0, 0, time.UTC),
			CloseTime: time.Date(2020, 4, 14, 16, 0, 0, 0, time.UTC),
		}, candle)
	//it's refresh time
	now = time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC)
	candles.refreshAll()
	//the minute candle is closed
	require.Len(t, updates, 2, "1 for old closed minute candle and 1 for the new empty minute candle")
	candle, ok = <-updates
	assert.True(t, ok)
	assert.Equal(t,
		CurrentCandle{
			Symbol:    "ETH/BTC",
			Open:      0.019,
			High:      0.019,
			Low:       0.019,
			Close:     0.019,
			Volume:    14.9,
			OpenTime:  time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
			CloseTime: time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
		}, candle)
	//new minute candle
	candle, ok = <-updates
	assert.True(t, ok)
	assert.Equal(t,
		CurrentCandle{
			Symbol:    "ETH/BTC",
			Open:      0,
			High:      0,
			Low:       0,
			Close:     0,
			Volume:    0,
			OpenTime:  time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
			CloseTime: time.Date(2020, 4, 14, 15, 47, 0, 0, time.UTC),
		}, candle)
	require.Len(t, updates, 0)
}
