package domain

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
	"time"
)

func TestChartToCurrentCandle(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		candle, err := ChartToCurrentCandle(nil, Candle1HResolution)
		require.NoError(t, err)
		assert.Equal(t, Candle{}, candle)
	})
	t.Run("one-sized", func(t *testing.T) {
		candle, err := ChartToCurrentCandle(&Chart{
			Symbol:     "0b2xJe",
			resolution: Candle1HResolution,
			O:          []primitive.Decimal128{mustParseDecimal128(t, "680.99")},
			H:          []primitive.Decimal128{mustParseDecimal128(t, "270.27")},
			L:          []primitive.Decimal128{mustParseDecimal128(t, "939.21")},
			C:          []primitive.Decimal128{mustParseDecimal128(t, "282.29")},
			V:          []primitive.Decimal128{mustParseDecimal128(t, "908.63")},
			T:          []int64{time.Date(2010, 1, 1, 14, 30, 0, 0, time.UTC).Unix()},
		}, Candle1HResolution)
		require.NoError(t, err)
		assert.Equal(t, Candle{
			Symbol:    "0b2xJe",
			Open:      mustParseDecimal128(t, "680.99"),
			High:      mustParseDecimal128(t, "270.27"),
			Low:       mustParseDecimal128(t, "939.21"),
			Close:     mustParseDecimal128(t, "282.29"),
			Volume:    mustParseDecimal128(t, "908.63"),
			OpenTime:  time.Date(2010, 1, 1, 14, 30, 0, 0, time.UTC),
			CloseTime: time.Date(2010, 1, 1, 15, 30, 0, 0, time.UTC),
		}, candle)
	})
	t.Run("3-sized", func(t *testing.T) {
		candle, err := ChartToCurrentCandle(&Chart{
			Symbol:     "0b2xJe",
			resolution: Candle1HResolution,
			O:          []primitive.Decimal128{mustParseDecimal128(t, "471.16"), mustParseDecimal128(t, "574.92"), mustParseDecimal128(t, "84.67")},
			H:          []primitive.Decimal128{mustParseDecimal128(t, "503.07"), mustParseDecimal128(t, "313.81"), mustParseDecimal128(t, "163.13")},
			L:          []primitive.Decimal128{mustParseDecimal128(t, "750.71"), mustParseDecimal128(t, "451.69"), mustParseDecimal128(t, "816.42")},
			C:          []primitive.Decimal128{mustParseDecimal128(t, "780.20"), mustParseDecimal128(t, "515.85"), mustParseDecimal128(t, "146.68")},
			V:          []primitive.Decimal128{mustParseDecimal128(t, "332.06"), mustParseDecimal128(t, "933.47"), mustParseDecimal128(t, "368.79")},
			T: []int64{
				time.Date(2010, 1, 1, 14, 30, 0, 0, time.UTC).Unix(),
				time.Date(2010, 1, 1, 15, 30, 0, 0, time.UTC).Unix(),
				time.Date(2010, 1, 1, 16, 30, 0, 0, time.UTC).Unix(),
			},
		}, Candle1HResolution)
		require.NoError(t, err)
		assert.Equal(t, Candle{
			Symbol:    "0b2xJe",
			Open:      mustParseDecimal128(t, "84.67"),
			High:      mustParseDecimal128(t, "163.13"),
			Low:       mustParseDecimal128(t, "816.42"),
			Close:     mustParseDecimal128(t, "146.68"),
			Volume:    mustParseDecimal128(t, "368.79"),
			OpenTime:  time.Date(2010, 1, 1, 16, 30, 0, 0, time.UTC),
			CloseTime: time.Date(2010, 1, 1, 17, 30, 0, 0, time.UTC),
		}, candle)
	})
}

func mustParseDecimal128(t *testing.T, s string) primitive.Decimal128 {
	decimal128, err := primitive.ParseDecimal128(s)
	require.NoError(t, err)
	return decimal128
}
