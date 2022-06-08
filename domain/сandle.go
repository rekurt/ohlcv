package domain

import (
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Candle struct {
	Symbol    string               `json:"symbol"`
	Open      primitive.Decimal128 `json:"o"`
	High      primitive.Decimal128 `json:"h"`
	Low       primitive.Decimal128 `json:"l"`
	Close     primitive.Decimal128 `json:"c"`
	Volume    primitive.Decimal128 `json:"v"`
	OpenTime  time.Time            `json:"t"`
	CloseTime time.Time
}

func (c Candle) ContainsTs(nano int64) bool {
	return c.OpenTime.UnixNano() <= nano && c.CloseTime.UnixNano() > nano
}

type Chart struct {
	Symbol     string `json:"symbol"`
	resolution string
	O          []primitive.Decimal128 `json:"o"`
	H          []primitive.Decimal128 `json:"h"`
	L          []primitive.Decimal128 `json:"l"`
	C          []primitive.Decimal128 `json:"c"`
	V          []primitive.Decimal128 `json:"v"`
	T          []int64                `json:"t"`
}

type ChartResponse struct {
	Symbol     string `json:"symbol"`
	resolution string
	O          []float64 `json:"o"`
	H          []float64 `json:"h"`
	L          []float64 `json:"l"`
	C          []float64 `json:"c"`
	V          []float64 `json:"v"`
	T          []int64   `json:"t"`
}

func (c *Chart) Resolution() string {
	return c.resolution
}

func (c *Chart) SetResolution(resolution string) {
	c.resolution = resolution
}

func (c *Chart) Market() string {
	return c.Symbol
}

func (c *Chart) SetMarket(market string) {
	c.Symbol = market
}

func MakeChartResponse(market string, chart *Chart) ChartResponse {
	if nil == chart {
		return ChartResponse{
			Symbol: market,
		}
	}

	r := ChartResponse{
		Symbol: chart.Symbol,
		H:      make([]float64, len(chart.H)),
		L:      make([]float64, len(chart.L)),
		O:      make([]float64, len(chart.O)),
		C:      make([]float64, len(chart.C)),
		V:      make([]float64, len(chart.V)),
		T:      chart.T,
	}

	for i := 0; i < len(chart.V); i++ {
		oString := chart.O[i].String()
		oFloat, _ := strconv.ParseFloat(oString, 64)
		hFloat, _ := strconv.ParseFloat(chart.H[i].String(), 64)
		lFloat, _ := strconv.ParseFloat(chart.L[i].String(), 64)
		cFloat, _ := strconv.ParseFloat(chart.C[i].String(), 64)
		vFloat, _ := strconv.ParseFloat(chart.V[i].String(), 64)

		r.O[i] = oFloat
		r.H[i] = hFloat
		r.L[i] = lFloat
		r.C[i] = cFloat
		r.V[i] = vFloat
	}

	return r
}

func ChartToCurrentCandle(chart *Chart, resolution string) (Candle, error) {
	if chart == nil {
		return Candle{}, nil
	}
	if len(chart.O) == 0 {
		return Candle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.O))
	}
	if len(chart.H) == 0 {
		return Candle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.H))
	}
	if len(chart.L) == 0 {
		return Candle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.L))
	}
	if len(chart.C) == 0 {
		return Candle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.C))
	}
	if len(chart.V) == 0 {
		return Candle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.V))
	}
	if len(chart.T) == 0 {
		return Candle{}, fmt.Errorf("unexpected len of chart: %d", len(chart.T))
	}
	openTime := time.Unix(chart.T[len(chart.T)-1], 0).UTC()
	return Candle{
		Symbol:    chart.Symbol,
		Open:      chart.O[len(chart.O)-1],
		High:      chart.H[len(chart.H)-1],
		Low:       chart.L[len(chart.L)-1],
		Close:     chart.C[len(chart.C)-1],
		Volume:    chart.V[len(chart.V)-1],
		OpenTime:  openTime,
		CloseTime: openTime.Add(StrResolutionToDuration(resolution)).UTC(),
	}, nil
}
