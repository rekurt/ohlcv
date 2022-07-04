package domain

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
	"time"
)

func TestMicrobatching(t *testing.T) {
	t.Parallel()
	t.Run("many full batches and 1 short", func(t *testing.T) {
		chartStream := make(chan *Chart)
		maxBatchSize := 13
		batchStream := Microbatching(context.Background(), chartStream, maxBatchSize)

		go func() {
			defer close(chartStream)
			for i := 0; i < 2378; i++ {
				chartStream <- &Chart{}
			}
		}()

		var batches [][]*Chart
		for charts := range batchStream {
			batches = append(batches, charts)
		}
		assert.Len(t, batches, 183)
		var i int
		for ; i < 182; i++ {
			assert.Len(t, batches[i], maxBatchSize)
		}
		assert.Len(t, batches[i], 12)
	})
	t.Run("2 seconds wait", func(t *testing.T) {
		chartStream := make(chan *Chart)
		maxBatchSize := 13
		batchStream := Microbatching(context.Background(), chartStream, maxBatchSize)

		go func() {
			for i := 0; i < 3; i++ {
				chartStream <- &Chart{}
			}
		}()

		started := time.Now()
		<-batchStream
		assert.InDelta(t, time.Second*2, time.Since(started), float64(time.Millisecond*200))
	})
}

func Test_mergeSameChart(t *testing.T) {
	t.Run("same symbol and resolution", func(t *testing.T) {
		batch := mergeSameChart([]*Chart{
			{
				Symbol:     "BTC",
				Resolution: "1min",
				O:          []primitive.Decimal128{mustParseDecimal128(t, "538.81")},
				H:          []primitive.Decimal128{mustParseDecimal128(t, "273.97")},
				L:          []primitive.Decimal128{mustParseDecimal128(t, "269.92")},
				C:          []primitive.Decimal128{mustParseDecimal128(t, "909.56")},
				V:          []primitive.Decimal128{mustParseDecimal128(t, "711.31")},
				T:          []int64{time.Date(2020, 1, 20, 0, 00, 0, 0, time.Local).Unix()},
			},
			{
				Symbol:     "BTC",
				Resolution: "1min",
				O:          []primitive.Decimal128{mustParseDecimal128(t, "453.05")},
				H:          []primitive.Decimal128{mustParseDecimal128(t, "952.78")},
				L:          []primitive.Decimal128{mustParseDecimal128(t, "402.97")},
				C:          []primitive.Decimal128{mustParseDecimal128(t, "599.34")},
				V:          []primitive.Decimal128{mustParseDecimal128(t, "665.45")},
				T:          []int64{time.Date(2020, 1, 20, 0, 01, 0, 0, time.Local).Unix()},
			},
		})
		assert.Equal(
			t,
			[]*Chart{
				{
					Symbol:     "BTC",
					Resolution: "1min",
					O:          []primitive.Decimal128{mustParseDecimal128(t, "538.81"), mustParseDecimal128(t, "453.05")},
					H:          []primitive.Decimal128{mustParseDecimal128(t, "273.97"), mustParseDecimal128(t, "952.78")},
					L:          []primitive.Decimal128{mustParseDecimal128(t, "269.92"), mustParseDecimal128(t, "402.97")},
					C:          []primitive.Decimal128{mustParseDecimal128(t, "909.56"), mustParseDecimal128(t, "599.34")},
					V:          []primitive.Decimal128{mustParseDecimal128(t, "711.31"), mustParseDecimal128(t, "665.45")},
					T: []int64{
						time.Date(2020, 1, 20, 0, 00, 0, 0, time.Local).Unix(),
						time.Date(2020, 1, 20, 0, 01, 0, 0, time.Local).Unix(),
					},
				},
			},
			batch,
		)
	})
	t.Run("no same symbol and resolution", func(t *testing.T) {
		batch := mergeSameChart([]*Chart{
			{
				Symbol:     "BTC",
				Resolution: "1min",
				O:          []primitive.Decimal128{mustParseDecimal128(t, "538.81")},
				H:          []primitive.Decimal128{mustParseDecimal128(t, "273.97")},
				L:          []primitive.Decimal128{mustParseDecimal128(t, "269.92")},
				C:          []primitive.Decimal128{mustParseDecimal128(t, "909.56")},
				V:          []primitive.Decimal128{mustParseDecimal128(t, "711.31")},
				T:          []int64{time.Date(2020, 1, 20, 0, 00, 0, 0, time.Local).Unix()},
			},
			{
				Symbol:     "BTC",
				Resolution: "2min",
				O:          []primitive.Decimal128{mustParseDecimal128(t, "453.05")},
				H:          []primitive.Decimal128{mustParseDecimal128(t, "952.78")},
				L:          []primitive.Decimal128{mustParseDecimal128(t, "402.97")},
				C:          []primitive.Decimal128{mustParseDecimal128(t, "599.34")},
				V:          []primitive.Decimal128{mustParseDecimal128(t, "665.45")},
				T:          []int64{time.Date(2020, 1, 20, 0, 01, 0, 0, time.Local).Unix()},
			},
		})
		assert.Equal(
			t,
			[]*Chart{
				{
					Symbol:     "BTC",
					Resolution: "1min",
					O:          []primitive.Decimal128{mustParseDecimal128(t, "538.81")},
					H:          []primitive.Decimal128{mustParseDecimal128(t, "273.97")},
					L:          []primitive.Decimal128{mustParseDecimal128(t, "269.92")},
					C:          []primitive.Decimal128{mustParseDecimal128(t, "909.56")},
					V:          []primitive.Decimal128{mustParseDecimal128(t, "711.31")},
					T:          []int64{time.Date(2020, 1, 20, 0, 00, 0, 0, time.Local).Unix()},
				},
				{
					Symbol:     "BTC",
					Resolution: "2min",
					O:          []primitive.Decimal128{mustParseDecimal128(t, "453.05")},
					H:          []primitive.Decimal128{mustParseDecimal128(t, "952.78")},
					L:          []primitive.Decimal128{mustParseDecimal128(t, "402.97")},
					C:          []primitive.Decimal128{mustParseDecimal128(t, "599.34")},
					V:          []primitive.Decimal128{mustParseDecimal128(t, "665.45")},
					T:          []int64{time.Date(2020, 1, 20, 0, 01, 0, 0, time.Local).Unix()},
				},
			},
			batch,
		)
	})
	t.Run("single chart", func(t *testing.T) {
		batch := mergeSameChart([]*Chart{
			{
				Symbol:     "BTC",
				Resolution: "1min",
				O:          []primitive.Decimal128{mustParseDecimal128(t, "538.81")},
				H:          []primitive.Decimal128{mustParseDecimal128(t, "273.97")},
				L:          []primitive.Decimal128{mustParseDecimal128(t, "269.92")},
				C:          []primitive.Decimal128{mustParseDecimal128(t, "909.56")},
				V:          []primitive.Decimal128{mustParseDecimal128(t, "711.31")},
				T:          []int64{time.Date(2020, 1, 20, 0, 00, 0, 0, time.Local).Unix()},
			},
		})
		assert.Equal(
			t,
			[]*Chart{
				{
					Symbol:     "BTC",
					Resolution: "1min",
					O:          []primitive.Decimal128{mustParseDecimal128(t, "538.81")},
					H:          []primitive.Decimal128{mustParseDecimal128(t, "273.97")},
					L:          []primitive.Decimal128{mustParseDecimal128(t, "269.92")},
					C:          []primitive.Decimal128{mustParseDecimal128(t, "909.56")},
					V:          []primitive.Decimal128{mustParseDecimal128(t, "711.31")},
					T:          []int64{time.Date(2020, 1, 20, 0, 00, 0, 0, time.Local).Unix()},
				},
			},
			batch,
		)
	})
}
