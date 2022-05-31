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
	go func() {
		for range candles.GetUpdates() {
			//to not block writer
		}
	}()
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
