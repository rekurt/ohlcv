package candle

import (
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"testing"
	"time"
)

func TestService_AggregateCandleToChartByResolution(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()
	mt.Run("success", func(mt *mtest.T) {
		s := Agregator{}
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

func  TestService_getMinuteCurrentTs(t *testing.T)  {
	tm := time.Unix(1650964257, 0)
	startMinuteTs := getStartMinuteTs(tm, 3)
	resultMinuteTime := time.Unix(startMinuteTs, 0)
	println(resultMinuteTime.Format("RFC850"))

	startHourTs := getStartHourTs(tm, 3)
	resultHourTime := time.Unix(startHourTs, 0)
	println(resultHourTime.Format("RFC850"))

	startMonthTs := getStartMonthTs(tm, 3)
	resultMonthTime := time.Unix(startMonthTs, 0)
	println(resultMonthTime.Format("RFC850"))
}
