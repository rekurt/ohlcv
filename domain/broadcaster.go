package domain

import (
	"context"
)

type ChartChannel struct {
	Name       string
	Market     string
	Resolution string
}

const CandleChartChannelPrefix = "candle_chart"

type Broadcaster interface {
	BroadcastCandleCharts(ctx context.Context, cht []*Chart)
}
