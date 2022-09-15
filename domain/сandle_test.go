package domain

import (
	"bitbucket.org/novatechnologies/ohlcv/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
	"time"
)

func TestChartToCurrentCandle(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		candle, err := ChartToCurrentCandle(nil, model.Candle1HResolution)
		require.NoError(t, err)
		assert.Equal(t, Candle{}, candle)
	})
	t.Run("one-sized", func(t *testing.T) {
		candle, err := ChartToCurrentCandle(&Chart{
			Symbol:     "0b2xJe",
			Resolution: model.Candle1HResolution,
			O:          []primitive.Decimal128{mustParseDecimal128(t, "680.99")},
			H:          []primitive.Decimal128{mustParseDecimal128(t, "270.27")},
			L:          []primitive.Decimal128{mustParseDecimal128(t, "939.21")},
			C:          []primitive.Decimal128{mustParseDecimal128(t, "282.29")},
			V:          []primitive.Decimal128{mustParseDecimal128(t, "908.63")},
			T:          []int64{time.Date(2010, 1, 1, 14, 30, 0, 0, time.UTC).Unix()},
		}, model.Candle1HResolution)
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
			Resolution: model.Candle1HResolution,
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
		}, model.Candle1HResolution)
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

func TestCandle_ContainsTs(t *testing.T) {
	type fields struct {
		Symbol     string
		Resolution string
		Open       primitive.Decimal128
		High       primitive.Decimal128
		Low        primitive.Decimal128
		Close      primitive.Decimal128
		Volume     primitive.Decimal128
		OpenTime   time.Time
		CloseTime  time.Time
	}
	type args struct {
		nano int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "contains",
			fields: fields{
				OpenTime:  time.Date(2019, 1, 25, 14, 10, 0, 0, time.UTC),
				CloseTime: time.Date(2021, 1, 25, 14, 10, 0, 0, time.UTC),
			},
			args: args{
				time.Date(2020, 1, 25, 14, 10, 0, 0, time.UTC).UnixNano(),
			},
			want: true,
		},
		{
			name: "before",
			fields: fields{
				OpenTime:  time.Date(2019, 1, 25, 14, 10, 0, 0, time.UTC),
				CloseTime: time.Date(2021, 1, 25, 14, 10, 0, 0, time.UTC),
			},
			args: args{
				time.Date(2019, 1, 24, 14, 10, 0, 0, time.UTC).UnixNano(),
			},
			want: false,
		},
		{
			name: "after",
			fields: fields{
				OpenTime:  time.Date(2019, 1, 25, 14, 10, 0, 0, time.UTC),
				CloseTime: time.Date(2021, 1, 25, 14, 10, 0, 0, time.UTC),
			},
			args: args{
				time.Date(2021, 1, 25, 14, 10, 0, 1, time.UTC).UnixNano(),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Candle{
				Symbol:     tt.fields.Symbol,
				Resolution: tt.fields.Resolution,
				Open:       tt.fields.Open,
				High:       tt.fields.High,
				Low:        tt.fields.Low,
				Close:      tt.fields.Close,
				Volume:     tt.fields.Volume,
				OpenTime:   tt.fields.OpenTime,
				CloseTime:  tt.fields.CloseTime,
			}
			assert.Equalf(t, tt.want, c.ContainsTs(tt.args.nano), "ContainsTs(%v)", tt.args.nano)
		})
	}
}
