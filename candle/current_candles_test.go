package candle

import (
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

func Test_updateCandle(t *testing.T) {
	candle, err := updateCandle(domain.Candle{}, matcher.Deal{Price: "866.13", Amount: "710.47"})
	require.NoError(t, err)
	assert.Equal(t, domain.Candle{
		Open:   mustParseDecimal128(t, "866.13"),
		High:   mustParseDecimal128(t, "866.13"),
		Low:    mustParseDecimal128(t, "866.13"),
		Close:  mustParseDecimal128(t, "866.13"),
		Volume: mustParseDecimal128(t, "710.47"),
	}, candle)
	candle, err = updateCandle(candle, matcher.Deal{Price: "861.60", Amount: "153.78"})
	require.NoError(t, err)
	assert.Equal(t, domain.Candle{
		Open:   mustParseDecimal128(t, "866.13"),
		High:   mustParseDecimal128(t, "866.13"),
		Low:    mustParseDecimal128(t, "861.60"),
		Close:  mustParseDecimal128(t, "861.60"),
		Volume: mustParseDecimal128(t, "864.25"),
	}, candle)
}

func TestNewCurrentCandles_updates(t *testing.T) {
	t.Run("get last ohlc on refresh", func(t *testing.T) {
		now := time.Date(2020, 4, 14, 15, 45, 56, 0, time.UTC)
		timeNow = func() time.Time {
			return now
		}
		updatesStream := make(chan domain.Candle, 512)
		candles := NewCurrentCandles(context.Background(), updatesStream).(*currentCandles)
		//init with empty candles
		for _, market := range []string{"ETH/BTC"} {
			for _, resolution := range []string{domain.Candle1MResolution} {
				openTime := time.Unix((&Aggregator{}).GetResolutionStartTimestampByTime(resolution, timeNow()), 0).UTC()
				require.NoError(t, candles.AddCandle(market, resolution, domain.Candle{
					Symbol:     "ETH/BTC",
					Resolution: resolution,
					Open:       mustParseDecimal128(t, "444.15"),
					High:       mustParseDecimal128(t, "933.37"),
					Low:        mustParseDecimal128(t, "152.63"),
					Close:      mustParseDecimal128(t, "636.74"),
					Volume:     mustParseDecimal128(t, "159.39"),
					OpenTime:   openTime,
					CloseTime:  openTime.Add(domain.StrResolutionToDuration(resolution)).UTC(),
				}))
			}
		}
		//2 new candles after init.
		require.Len(t, updatesStream, 1)
		_, ok := <-updatesStream
		assert.True(t, ok)
		//it's refresh time
		now = time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC)
		candles.refreshAll()
		require.Len(t, updatesStream, 2, "1 for old closed minute candle and 1 for the new empty minute candle")
		candle, ok := <-updatesStream
		assert.True(t, ok)
		assert.Equal(t,
			domain.Candle{
				Symbol:     "ETH/BTC",
				Resolution: domain.Candle1MResolution,
				Open:       mustParseDecimal128(t, "444.15"),
				High:       mustParseDecimal128(t, "933.37"),
				Low:        mustParseDecimal128(t, "152.63"),
				Close:      mustParseDecimal128(t, "636.74"),
				Volume:     mustParseDecimal128(t, "159.39"),
				OpenTime:   time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
				CloseTime:  time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
			}, candle)
		//new minute candle
		candle, ok = <-updatesStream
		assert.True(t, ok)
		assert.Equal(t,
			domain.Candle{
				Symbol:     "ETH/BTC",
				Resolution: domain.Candle1MResolution,
				Open:       mustParseDecimal128(t, "636.74"),
				High:       mustParseDecimal128(t, "636.74"),
				Low:        mustParseDecimal128(t, "636.74"),
				Close:      mustParseDecimal128(t, "636.74"),
				Volume:     primitive.Decimal128{},
				OpenTime:   time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
				CloseTime:  time.Date(2020, 4, 14, 15, 47, 0, 0, time.UTC),
			}, candle, "inherit ohlc values from previous candle")
		require.Len(t, updatesStream, 0)
		//when first deal arrives
		require.NoError(t, candles.AddDeal(matcher.Deal{
			Market:    "ETH/BTC",
			CreatedAt: time.Date(2020, 4, 14, 15, 46, 53, 0, time.UTC).UnixNano(),
			Price:     "0.013",
			Amount:    "1.9",
		}))
		candle, ok = <-updatesStream
		assert.True(t, ok)
		assert.Equal(t,
			domain.Candle{
				Symbol:     "ETH/BTC",
				Resolution: domain.Candle1MResolution,
				Open:       mustParseDecimal128(t, "0.013"),
				High:       mustParseDecimal128(t, "636.74"),
				Low:        mustParseDecimal128(t, "0.013"),
				Close:      mustParseDecimal128(t, "0.013"),
				Volume:     mustParseDecimal128(t, "1.9"),
				OpenTime:   time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
				CloseTime:  time.Date(2020, 4, 14, 15, 47, 0, 0, time.UTC),
			},
			candle,
		)
		require.Len(t, updatesStream, 0)
	})
	t.Run("1 market 1 deal 2 resolutions", func(t *testing.T) {
		now := time.Date(2020, 4, 14, 15, 45, 56, 0, time.UTC)
		timeNow = func() time.Time {
			return now
		}
		updatesStream := make(chan domain.Candle, 512)
		candles := NewCurrentCandles(context.Background(), updatesStream).(*currentCandles)
		//init with empty candles
		for _, market := range []string{"ETH/BTC"} {
			for _, resolution := range []string{domain.Candle1MResolution, domain.Candle1HResolution} {
				require.NoError(t, candles.AddCandle(market, resolution, domain.Candle{}))
			}
		}
		//2 new candles after init.
		require.Len(t, updatesStream, 2)
		candle, ok := <-updatesStream
		assert.True(t, ok)
		assert.Equal(t,
			domain.Candle{
				Symbol:     "ETH/BTC",
				Resolution: domain.Candle1MResolution,
				OpenTime:   time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
				CloseTime:  time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
			}, candle)

		candle, ok = <-updatesStream
		assert.True(t, ok)
		assert.Equal(t,
			domain.Candle{
				Symbol:     "ETH/BTC",
				Resolution: domain.Candle1HResolution,
				OpenTime:   time.Date(2020, 4, 14, 15, 0, 0, 0, time.UTC),
				CloseTime:  time.Date(2020, 4, 14, 16, 0, 0, 0, time.UTC),
			}, candle)
		//make a deal
		require.NoError(t, candles.AddDeal(matcher.Deal{
			Market:    "ETH/BTC",
			CreatedAt: time.Date(2020, 4, 14, 15, 45, 50, 0, time.UTC).UnixNano(),
			Price:     "0.019",
			Amount:    "14.9",
		}))
		//both candles are updated
		require.Len(t, updatesStream, 2, "two new with the deal")
		//new minute candle with the deal
		candle, ok = <-updatesStream
		assert.True(t, ok)
		assert.Equal(t,
			domain.Candle{
				Symbol:     "ETH/BTC",
				Resolution: domain.Candle1MResolution,
				Open:       mustParseDecimal128(t, "0.019"),
				High:       mustParseDecimal128(t, "0.019"),
				Low:        mustParseDecimal128(t, "0.019"),
				Close:      mustParseDecimal128(t, "0.019"),
				Volume:     mustParseDecimal128(t, "14.9"),
				OpenTime:   time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
				CloseTime:  time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
			}, candle)
		//new hour candle with the deal
		candle, ok = <-updatesStream
		assert.True(t, ok)
		assert.Equal(t,
			domain.Candle{
				Symbol:     "ETH/BTC",
				Resolution: domain.Candle1HResolution,
				Open:       mustParseDecimal128(t, "0.019"),
				High:       mustParseDecimal128(t, "0.019"),
				Low:        mustParseDecimal128(t, "0.019"),
				Close:      mustParseDecimal128(t, "0.019"),
				Volume:     mustParseDecimal128(t, "14.9"),
				OpenTime:   time.Date(2020, 4, 14, 15, 0, 0, 0, time.UTC),
				CloseTime:  time.Date(2020, 4, 14, 16, 0, 0, 0, time.UTC),
			}, candle)
		//it's refresh time
		now = time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC)
		candles.refreshAll()
		//the minute candle is closed, but hour candle is not closed
		require.Len(t, updatesStream, 2, "1 for old closed minute candle and 1 for the new empty minute candle")
		candle, ok = <-updatesStream
		assert.True(t, ok)
		assert.Equal(t,
			domain.Candle{
				Symbol:     "ETH/BTC",
				Resolution: domain.Candle1MResolution,
				Open:       mustParseDecimal128(t, "0.019"),
				High:       mustParseDecimal128(t, "0.019"),
				Low:        mustParseDecimal128(t, "0.019"),
				Close:      mustParseDecimal128(t, "0.019"),
				Volume:     mustParseDecimal128(t, "14.9"),
				OpenTime:   time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
				CloseTime:  time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
			}, candle)
		//new minute candle
		candle, ok = <-updatesStream
		assert.True(t, ok)
		assert.Equal(t,
			domain.Candle{
				Symbol:     "ETH/BTC",
				Resolution: domain.Candle1MResolution,
				Open:       mustParseDecimal128(t, "0.019"),
				High:       mustParseDecimal128(t, "0.019"),
				Low:        mustParseDecimal128(t, "0.019"),
				Close:      mustParseDecimal128(t, "0.019"),
				OpenTime:   time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
				CloseTime:  time.Date(2020, 4, 14, 15, 47, 0, 0, time.UTC),
			}, candle, "inherit o h l c values from previous candle")
		require.Len(t, updatesStream, 0)
	})
	t.Run("1 market 2 deal 1 resolutions", func(t *testing.T) {
		now := time.Date(2020, 4, 14, 15, 45, 56, 0, time.UTC)
		timeNow = func() time.Time {
			return now
		}
		updatesStream := make(chan domain.Candle, 512)
		candles := NewCurrentCandles(context.Background(), updatesStream).(*currentCandles)
		//init with empty candles
		for _, market := range []string{"ETH/BTC"} {
			for _, resolution := range []string{domain.Candle1MResolution} {
				require.NoError(t, candles.AddCandle(market, resolution, domain.Candle{}))
			}
		}
		//1 new candles after init
		require.Len(t, updatesStream, 1)
		candle, ok := <-updatesStream
		assert.True(t, ok)
		assert.Equal(t,
			domain.Candle{
				Symbol:     "ETH/BTC",
				Resolution: domain.Candle1MResolution,
				OpenTime:   time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
				CloseTime:  time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
			}, candle)
		//make a deal
		require.NoError(t, candles.AddDeal(matcher.Deal{
			Market:    "ETH/BTC",
			CreatedAt: time.Date(2020, 4, 14, 15, 45, 50, 0, time.UTC).UnixNano(),
			Price:     "0.019",
			Amount:    "14.9",
		}))
		//minute candle is updated
		require.Len(t, updatesStream, 1)
		//new minute candle with the deal
		candle, ok = <-updatesStream
		assert.True(t, ok)
		assert.Equal(t,
			domain.Candle{
				Symbol:     "ETH/BTC",
				Resolution: domain.Candle1MResolution,
				Open:       mustParseDecimal128(t, "0.019"),
				High:       mustParseDecimal128(t, "0.019"),
				Low:        mustParseDecimal128(t, "0.019"),
				Close:      mustParseDecimal128(t, "0.019"),
				Volume:     mustParseDecimal128(t, "14.9"),
				OpenTime:   time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
				CloseTime:  time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
			}, candle)
		//make another deal
		require.NoError(t, candles.AddDeal(matcher.Deal{
			Market:    "ETH/BTC",
			CreatedAt: time.Date(2020, 4, 14, 15, 45, 53, 0, time.UTC).UnixNano(),
			Price:     "0.013",
			Amount:    "1.9",
		}))
		require.Len(t, updatesStream, 1)
		//new minute candle
		candle, ok = <-updatesStream
		assert.True(t, ok)
		assert.Equal(t,
			domain.Candle{
				Symbol:     "ETH/BTC",
				Resolution: domain.Candle1MResolution,
				Open:       mustParseDecimal128(t, "0.019"),
				High:       mustParseDecimal128(t, "0.019"),
				Low:        mustParseDecimal128(t, "0.013"),
				Close:      mustParseDecimal128(t, "0.013"),
				Volume:     mustParseDecimal128(t, "16.8"),
				OpenTime:   time.Date(2020, 4, 14, 15, 45, 0, 0, time.UTC),
				CloseTime:  time.Date(2020, 4, 14, 15, 46, 0, 0, time.UTC),
			}, candle)
		require.Len(t, updatesStream, 0)
		//make miss deal with a non-existent market
		require.NoError(t, candles.AddDeal(matcher.Deal{
			Market:    "ETH/CRONA",
			CreatedAt: time.Date(2020, 4, 14, 15, 45, 53, 0, time.UTC).UnixNano(),
			Price:     "0.013",
			Amount:    "1.9",
		}))
		//no updates
		require.Len(t, updatesStream, 0)
	})

}

//add logs to refreshAll to ensure it runs every round minute
/*
Output:
2022-06-02 11:32:00.000299 +0300 MSK m=+32.517239959 refreshAll
2022-06-02 11:33:00.000162 +0300 MSK m=+92.518844209 refreshAll
2022-06-02 11:34:00.011105 +0300 MSK m=+152.519722292 refreshAll
2022-06-02 11:35:00.013223 +0300 MSK m=+212.509831459 refreshAll
2022-06-02 11:36:00.013223 +0300 MSK m=+212.509831459 refreshAll
*/
func Test_everyMinute_manual(t *testing.T) {
	t.Skip()
	t.Run("regular", func(t *testing.T) {
		_ = NewCurrentCandles(context.Background(), nil)
		select {}
	})
}

func Test_concurrent(t *testing.T) {
	updatesStream := make(chan domain.Candle, 512)
	candles := NewCurrentCandles(context.Background(), updatesStream)
	markets := []string{"market1", "market2", "market3"}
	for _, market := range markets {
		for _, resolution := range []string{domain.Candle1MResolution, domain.Candle1HResolution, domain.Candle15MResolution} {
			openTime := time.Unix((&Aggregator{}).GetResolutionStartTimestampByTime(resolution, time.Now()), 0).UTC()
			require.NoError(t, candles.AddCandle(market, resolution, domain.Candle{
				Symbol:    market,
				OpenTime:  openTime,
				CloseTime: openTime.Add(domain.StrResolutionToDuration(resolution)).UTC(),
			}),
			)
		}
	}
	go func() {
		for range updatesStream {

		}
	}()
	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				require.NoError(t, candles.AddDeal(matcher.Deal{
					Market:    markets[rand.Intn(len(markets))],
					CreatedAt: time.Now().UnixNano(),
					Price:     strconv.FormatFloat(rand.Float64()*float64(rand.Intn(100)), 'f', 5, 64),
					Amount:    strconv.FormatFloat(rand.Float64()*float64(rand.Intn(100)), 'f', 5, 64),
				}))
			}
		}()
	}
	wg.Wait()
}
