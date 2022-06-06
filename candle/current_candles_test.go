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

func TestNewCurrentCandles_updates(t *testing.T) {
	t.Run("1 market 1 deal 2 resolutions", func(t *testing.T) {
		now := time.Date(2020, 4, 14, 15, 45, 56, 0, time.UTC)
		timeNow = func() time.Time {
			return now
		}
		candles := NewCurrentCandles(context.Background()).(*currentCandles)
		updates := candles.GetUpdates()
		//init with empty candles
		for _, market := range []string{"ETH/BTC"} {
			for _, resolution := range []string{domain.Candle1MResolution, domain.Candle1HResolution} {
				openTime := time.Unix((&Aggregator{}).GetCurrentResolutionStartTimestamp(resolution, now), 0).UTC()
				require.NoError(t, candles.AddCandle(market, resolution, CurrentCandle{
					Symbol:    "ETH/BTC",
					OpenTime:  openTime,
					CloseTime: openTime.Add(domain.StrResolutionToDuration(resolution)).UTC(),
				}))
			}
		}
		//2 new candles after init
		require.Len(t, updates, 2)
		candle, ok := <-updates
		assert.True(t, ok)
		assert.Equal(t,
			CurrentCandle{
				Symbol:    "ETH/BTC",
				OpenTime:  time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
				CloseTime: time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
			}, candle)

		candle, ok = <-updates
		assert.True(t, ok)
		assert.Equal(t,
			CurrentCandle{
				Symbol:    "ETH/BTC",
				OpenTime:  time.Date(2020, 4, 14, 15, 0, 0, 0, time.UTC),
				CloseTime: time.Date(2020, 4, 14, 16, 0, 0, 0, time.UTC),
			}, candle)
		//make a deal
		require.NoError(t, candles.AddDeal(matcher.Deal{
			Market:    "ETH/BTC",
			CreatedAt: time.Date(2020, 4, 14, 15, 45, 50, 0, time.UTC).UnixNano(),
			Price:     "0.019",
			Amount:    "14.9",
		}))
		//both candles are updated
		require.Len(t, updates, 4, "two empty old and two new with the deal")
		//old empty minute candle
		candle, ok = <-updates
		assert.True(t, ok)
		assert.Equal(t,
			CurrentCandle{
				Symbol:    "ETH/BTC",
				OpenTime:  time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
				CloseTime: time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
			}, candle)
		//new minute candle with the deal
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
		//old empty hour candle
		candle, ok = <-updates
		assert.True(t, ok)
		assert.Equal(t,
			CurrentCandle{
				Symbol:    "ETH/BTC",
				OpenTime:  time.Date(2020, 4, 14, 15, 0, 0, 0, time.UTC),
				CloseTime: time.Date(2020, 4, 14, 16, 0, 0, 0, time.UTC),
			}, candle)
		//new hour candle with the deal
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
		//the minute candle is closed, but hour candle is not closed
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
				OpenTime:  time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
				CloseTime: time.Date(2020, 4, 14, 15, 47, 0, 0, time.UTC),
			}, candle)
		require.Len(t, updates, 0)
	})
	t.Run("1 market 2 deal 1 resolutions", func(t *testing.T) {
		now := time.Date(2020, 4, 14, 15, 45, 56, 0, time.UTC)
		timeNow = func() time.Time {
			return now
		}
		candles := NewCurrentCandles(context.Background()).(*currentCandles)
		updates := candles.GetUpdates()
		//init with empty candles
		for _, market := range []string{"ETH/BTC"} {
			for _, resolution := range []string{domain.Candle1MResolution} {
				openTime := time.Unix((&Aggregator{}).GetCurrentResolutionStartTimestamp(resolution, now), 0).UTC()
				require.NoError(t, candles.AddCandle(market, resolution, CurrentCandle{
					Symbol:    "ETH/BTC",
					OpenTime:  openTime,
					CloseTime: openTime.Add(domain.StrResolutionToDuration(resolution)).UTC(),
				}))
			}
		}
		//1 new candles after init
		require.Len(t, updates, 1)
		candle, ok := <-updates
		assert.True(t, ok)
		assert.Equal(t,
			CurrentCandle{
				Symbol:    "ETH/BTC",
				OpenTime:  time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
				CloseTime: time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
			}, candle)
		//make a deal
		require.NoError(t, candles.AddDeal(matcher.Deal{
			Market:    "ETH/BTC",
			CreatedAt: time.Date(2020, 4, 14, 15, 45, 50, 0, time.UTC).UnixNano(),
			Price:     "0.019",
			Amount:    "14.9",
		}))
		//minute candle is updated
		require.Len(t, updates, 2)
		//old empty minute candle
		candle, ok = <-updates
		assert.True(t, ok)
		assert.Equal(t,
			CurrentCandle{
				Symbol:    "ETH/BTC",
				OpenTime:  time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
				CloseTime: time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
			}, candle)
		//new minute candle with the deal
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
		//make another deal
		require.NoError(t, candles.AddDeal(matcher.Deal{
			Market:    "ETH/BTC",
			CreatedAt: time.Date(2020, 4, 14, 15, 45, 53, 0, time.UTC).UnixNano(),
			Price:     "0.013",
			Amount:    "1.9",
		}))
		require.Len(t, updates, 2)
		//old minute candle
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
				Open:      0.019,
				High:      0.019,
				Low:       0.013,
				Close:     0.013,
				Volume:    14.9 + 1.9,
				OpenTime:  time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
				CloseTime: time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
			}, candle)
		require.Len(t, updates, 0)
		//make miss deal with a non-existent market
		require.NoError(t, candles.AddDeal(matcher.Deal{
			Market:    "ETH/CRONA",
			CreatedAt: time.Date(2020, 4, 14, 15, 45, 53, 0, time.UTC).UnixNano(),
			Price:     "0.013",
			Amount:    "1.9",
		}))
		//no updates
		require.Len(t, updates, 0)
	})

}

//add logs to refreshAll to ensure it runs every round minute
/*
Output:
2022-06-02 11:32:00.000299 +0300 MSK m=+32.517239959 refreshAll
2022-06-02 11:33:00.000162 +0300 MSK m=+92.518844209 refreshAll
2022-06-02 11:34:00.011105 +0300 MSK m=+152.519722292 refreshAll
2022-06-02 11:35:00.013223 +0300 MSK m=+212.509831459 refreshAll
*/
func Test_everyMinute_manual(t *testing.T) {
	t.Skip()
	t.Run("regular", func(t *testing.T) {
		_ = NewCurrentCandles(context.Background())
		select {}
	})
}
