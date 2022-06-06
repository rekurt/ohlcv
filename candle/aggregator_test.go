package candle

import (
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
		Open:      o1,
		High:      h1,
		Low:       l1,
		Close:     cl1,
		Volume:    v1,
		Timestamp: ts1,
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
