package candle

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra/centrifuge"
	"context"
	"encoding/json"
	"fmt"
)

type broadcaster struct {
	Centrifuge centrifuge.Centrifuge
	Channels   map[string]map[string]*domain.ChartChannel
}

func NewBroadcaster(centrifuge centrifuge.Centrifuge, channels map[string]map[string]*domain.ChartChannel) *broadcaster {
	return &broadcaster{Centrifuge: centrifuge, Channels: channels}
}

func (b broadcaster) BroadcastCandleCharts(ctx context.Context, cht []*domain.Chart) {
	messages := make([]centrifuge.MessageData, 0)

	for _, chart := range cht {
		logger.FromContext(ctx).WithField("market", chart.Market()).WithField("reolution", chart.Resolution()).Infof("[Broadcaster.BroadcastCandleCharts]Broadcasting charts.")

		channel := b.Channels[chart.Market()][chart.Resolution()]
		payload, _ := json.Marshal(chart)
		messages = append(messages, centrifuge.MessageData{
			Channel: channel.Name,
			Data:    string(payload),
		})
	}

	logger.FromContext(ctx).WithField("messageCount", len(messages)).Infof("[Broadcaster.BroadcastCandleCharts]Push charts to centrifuge.")
	b.Centrifuge.BatchPublish(ctx, messages)
}

func GetChartsChannels() map[string]map[string]*domain.ChartChannel {
	m := domain.GetAvailableMarkets()
	c := make(map[string]map[string]*domain.ChartChannel, len(m))
	for _, market := range m {
		resolutions := domain.GetAvailableResolutions()
		marketChannels := make(map[string]*domain.ChartChannel, len(resolutions))
		for _, resolution := range resolutions {
			marketChannels[resolution] = NewChartChannel(market, resolution)
		}
		c[market] = marketChannels
	}
	return c
}

func NewChartChannel(market string, resolution string) *domain.ChartChannel {
	name := fmt.Sprintf("%s_%s_%s", domain.CandleChartChannelPrefix, market, resolution)
	return &domain.ChartChannel{
		Name:       name,
		Market:     market,
		Resolution: resolution,
	}
}

