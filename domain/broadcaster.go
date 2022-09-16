package domain

import (
	"bitbucket.org/novatechnologies/ohlcv/internal/model"
	"context"
)

type ChartChannel struct {
	Name       string
	Market     string
	Resolution model.Resolution
}

const CandleChartChannelPrefix = "candle_chart"

type Broadcaster interface {
	BroadcastCandleCharts(ctx context.Context, cht []*Chart)
}
