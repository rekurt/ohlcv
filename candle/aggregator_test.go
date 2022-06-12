package candle

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"bitbucket.org/novatechnologies/ohlcv/domain"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestService_AggregateCandleToChartByResolution(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()
	mt.Run("success", func(mt *mtest.T) {
		s := Aggregator{}
		market := "BTC/USDT"
		cs := getCandles()
		chart := s.AggregateCandleToChartByResolution(cs, market, domain.Candle5MResolution, 0)
		assert.Len(t, chart, 1)
	})
}

func getCandles() []*domain.Candle {
	return []*domain.Candle{
		generateCandle("58,345", "58,615", "58,205", "58,245", "600", 165088495115),
	}
}
func BenchmarkName(b *testing.B) {
	for i := 0; i < b.N; i++ {

	}
}
func generateCandle(o string, h string, l string, cl string, v string, ts int64) *domain.Candle {
	o1, _ := primitive.ParseDecimal128(o)
	h1, _ := primitive.ParseDecimal128(h)
	l1, _ := primitive.ParseDecimal128(l)
	cl1, _ := primitive.ParseDecimal128(cl)
	v1, _ := primitive.ParseDecimal128(v)
	ts1 := time.Unix(ts, 0)

	return &domain.Candle{
		Open:     o1,
		High:     h1,
		Low:      l1,
		Close:    cl1,
		Volume:   v1,
		OpenTime: ts1,
	}
}

func TestService_getMinuteCurrentTs(t *testing.T) {
	t.Run("1 min", func(t *testing.T) {
		now, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
		require.NoError(t, err)
		assert.Equal(t,
			"2006-01-02T15:04:00Z",
			time.Unix(getStartMinuteTs(now, 1), 0).UTC().Format(time.RFC3339),
		)
	})
	t.Run("3 min", func(t *testing.T) {
		now, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
		require.NoError(t, err)
		assert.Equal(t,
			"2006-01-02T15:03:00Z",
			time.Unix(getStartMinuteTs(now, 3), 0).UTC().Format(time.RFC3339),
		)
	})
	t.Run("5 min", func(t *testing.T) {
		now, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
		require.NoError(t, err)
		assert.Equal(t,
			"2006-01-02T15:00:00Z",
			time.Unix(getStartMinuteTs(now, 5), 0).UTC().Format(time.RFC3339),
		)
	})
	t.Run("30 min", func(t *testing.T) {
		now, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
		require.NoError(t, err)
		assert.Equal(t,
			"2006-01-02T15:00:00Z",
			time.Unix(getStartMinuteTs(now, 30), 0).UTC().Format(time.RFC3339),
		)
	})
}

func TestService_getStartHourTs(t *testing.T) {
	t.Run("1 hour", func(t *testing.T) {
		now, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
		require.NoError(t, err)
		assert.Equal(t,
			"2006-01-02T15:00:00Z",
			time.Unix(getStartHourTs(now, 1), 0).UTC().Format(time.RFC3339),
		)
	})
	t.Run("2 hour", func(t *testing.T) {
		now, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
		require.NoError(t, err)
		assert.Equal(t,
			"2006-01-02T14:00:00Z",
			time.Unix(getStartHourTs(now, 2), 0).UTC().Format(time.RFC3339),
		)
	})
	t.Run("24 hour", func(t *testing.T) {
		now, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
		require.NoError(t, err)
		assert.Equal(t,
			"2006-01-02T00:00:00Z",
			time.Unix(getStartHourTs(now, 24), 0).UTC().Format(time.RFC3339),
		)
	})
}

func Test_compareDecimal128(t *testing.T) {
	type args struct {
		d1 primitive.Decimal128
		d2 primitive.Decimal128
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "first bigger",
			args: args{
				d1: mustParseDecimal128(t, "374"),
				d2: mustParseDecimal128(t, "130"),
			},
			want: 1,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return err == nil
			},
		},
		{
			name: "first bigger one negative",
			args: args{
				d1: mustParseDecimal128(t, "374"),
				d2: mustParseDecimal128(t, "-130.6543"),
			},
			want: 1,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return err == nil
			},
		},
		//{ TODO fix bug that fails this test
		//	name: "first bigger both negative",
		//	args: args{
		//		d1: mustParseDecimal128(t, "-4"),
		//		d2: mustParseDecimal128(t, "-130.6543"),
		//	},
		//	want: 1,
		//	wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
		//		return err == nil
		//	},
		//},
		{
			name: "second bigger",
			args: args{
				d1: mustParseDecimal128(t, "374"),
				d2: mustParseDecimal128(t, "861"),
			},
			want: -1,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return err == nil
			},
		},
		{
			name: "second bigger, one negative",
			args: args{
				d1: mustParseDecimal128(t, "-374"),
				d2: mustParseDecimal128(t, "861"),
			},
			want: -1,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return err == nil
			},
		},
		{
			name: "same",
			args: args{
				d1: mustParseDecimal128(t, "67"),
				d2: mustParseDecimal128(t, "67"),
			},
			want: 0,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return err == nil
			},
		},
		{
			name: "same with decimal part",
			args: args{
				d1: mustParseDecimal128(t, "67.98496867"),
				d2: mustParseDecimal128(t, "67.98496867"),
			},
			want: 0,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return err == nil
			},
		},
		{
			name: "same with decimal part and negative",
			args: args{
				d1: mustParseDecimal128(t, "-67.98496867"),
				d2: mustParseDecimal128(t, "-67.98496867"),
			},
			want: 0,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return err == nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := compareDecimal128(tt.args.d1, tt.args.d2)
			if !tt.wantErr(t, err, fmt.Sprintf("compareDecimal128(%v, %v)", tt.args.d1, tt.args.d2)) {
				return
			}
			assert.Equalf(t, tt.want, got, "compareDecimal128(%v, %v)", tt.args.d1, tt.args.d2)
		})
	}
}

func Test_addPrimitiveDecimal128(t *testing.T) {
	type args struct {
		a primitive.Decimal128
		b primitive.Decimal128
	}
	tests := []struct {
		name    string
		args    args
		want    primitive.Decimal128
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "add with zero",
			args: args{
				a: mustParseDecimal128(t, "916.88243"),
				b: mustParseDecimal128(t, "0"),
			},
			want: mustParseDecimal128(t, "916.88243"),
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return err == nil
			},
		},
		{
			name: "add with negative zero",
			args: args{
				a: mustParseDecimal128(t, "916.88243"),
				b: mustParseDecimal128(t, "-0"),
			},
			want: mustParseDecimal128(t, "916.88243"),
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return err == nil
			},
		},
		{
			name: "add with negative zero with decimal part",
			args: args{
				a: mustParseDecimal128(t, "916.88243"),
				b: mustParseDecimal128(t, "-0.00"),
			},
			want: mustParseDecimal128(t, "916.88243"),
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return err == nil
			},
		},
		{
			name: "add positive",
			args: args{
				a: mustParseDecimal128(t, "916.88243"),
				b: mustParseDecimal128(t, "6543"),
			},
			want: mustParseDecimal128(t, "7459.88243"),
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return err == nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := addPrimitiveDecimal128(tt.args.a, tt.args.b)
			if !tt.wantErr(t, err, fmt.Sprintf("addPrimitiveDecimal128(%v, %v)", tt.args.a, tt.args.b)) {
				return
			}
			assert.Equalf(t, tt.want, got, "addPrimitiveDecimal128(%v, %v)", tt.args.a, tt.args.b)
		})
	}
}

func mustParseDecimal128(t *testing.T, s string) primitive.Decimal128 {
	decimal128, err := primitive.ParseDecimal128(s)
	require.NoError(t, err)
	return decimal128
}

func TestAggregator_aggregateHoursCandlesToChart(t *testing.T) {
	agg := &Aggregator{}
	chart := agg.aggregateHoursCandlesToChart([]*domain.Candle{
		{
			Symbol:    "ETH-BTC",
			Open:      mustParseDecimal128(t, "538.81"),
			High:      mustParseDecimal128(t, "273.97"),
			Low:       mustParseDecimal128(t, "269.92"),
			Close:     mustParseDecimal128(t, "909.56"),
			Volume:    mustParseDecimal128(t, "711.31"),
			OpenTime:  time.Date(2020, 1, 20, 13, 45, 0, 0, time.UTC),
			CloseTime: time.Date(2020, 1, 20, 16, 45, 0, 0, time.UTC),
		},
	}, 1)
	assert.Equal(t, &domain.Chart{
		Symbol: "",
		O:      []primitive.Decimal128{mustParseDecimal128(t, "538.81")},
		H:      []primitive.Decimal128{mustParseDecimal128(t, "273.97")},
		L:      []primitive.Decimal128{mustParseDecimal128(t, "269.92")},
		C:      []primitive.Decimal128{mustParseDecimal128(t, "909.56")},
		V:      []primitive.Decimal128{mustParseDecimal128(t, "711.31")},
		T:      []int64{time.Date(2020, 1, 20, 13, 00, 0, 0, time.UTC).Unix()},
	}, chart)
}
func TestAggregator_aggregateWeekCandlesToChart(t *testing.T) {
	agg := &Aggregator{}
	chart := agg.aggregateWeekCandlesToChart([]*domain.Candle{
		{
			Symbol:    "ETH-BTC",
			Open:      mustParseDecimal128(t, "538.81"),
			High:      mustParseDecimal128(t, "273.97"),
			Low:       mustParseDecimal128(t, "269.92"),
			Close:     mustParseDecimal128(t, "909.56"),
			Volume:    mustParseDecimal128(t, "711.31"),
			OpenTime:  time.Date(2020, 1, 20, 13, 45, 0, 0, time.Local),
			CloseTime: time.Date(2020, 1, 20, 16, 45, 0, 0, time.Local),
		},
	})
	assert.Equal(t, &domain.Chart{
		Symbol: "",
		O:      []primitive.Decimal128{mustParseDecimal128(t, "538.81")},
		H:      []primitive.Decimal128{mustParseDecimal128(t, "273.97")},
		L:      []primitive.Decimal128{mustParseDecimal128(t, "269.92")},
		C:      []primitive.Decimal128{mustParseDecimal128(t, "909.56")},
		V:      []primitive.Decimal128{mustParseDecimal128(t, "711.31")},
		T:      []int64{time.Date(2020, 1, 20, 0, 00, 0, 0, time.Local).Unix()},
	}, chart)
}

func Test_firstDayOfISOWeek(t *testing.T) {
	type args struct {
		year     int
		week     int
		timezone *time.Location
	}
	tests := []struct {
		name string
		args args
		want time.Time
	}{
		{
			name: "10 June",
			args: args{
				year:     2022,
				week:     23,
				timezone: time.UTC,
			},
			want: time.Date(2022, 6, 6, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, firstDayOfISOWeek(tt.args.year, tt.args.week, tt.args.timezone), "firstDayOfISOWeek(%v, %v, %v)", tt.args.year, tt.args.week, tt.args.timezone)
		})
	}
}
