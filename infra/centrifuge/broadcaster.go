package centrifuge

import (
	"context"
	"encoding/json"
	"fmt"

	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/domain"
)

type broadcaster struct {
	Centrifuge   Centrifuge
	Channels     map[string]map[string]*domain.ChartChannel
	eventsBroker domain.EventsBroker
}

func NewBroadcaster(
	publisher Centrifuge,
	eventsBroker domain.EventsBroker,
) *broadcaster {
	return &broadcaster{
		Centrifuge:   publisher,
		Channels:     GetChartsChannels(),
		eventsBroker: eventsBroker,
	}
}

func (b broadcaster) SubscribeForCharts() {
	b.eventsBroker.Subscribe(
		domain.EvTypeCharts, func(e *domain.Event) error {
			b.BroadcastCandleCharts(e.Ctx, e.MustGetCharts())
			return nil
		},
	)
}

func (b broadcaster) BroadcastCandleCharts(
	ctx context.Context,
	cht []*domain.Chart,
) {
	messages := make([]MessageData, 0)

	for _, chart := range cht {
		channel := b.Channels[chart.Market()][chart.Resolution()]
		payload, _ := json.Marshal(chart)
		messages = append(
			messages, MessageData{
				Channel: channel.Name,
				Data:    string(payload),
			},
		)
	}

	logger.FromContext(ctx).WithField(
		"messageCount",
		len(messages),
	).WithField(
		"messages",
		messages,
	).Infof("[Broadcaster.BroadcastCandleCharts] Push charts to Centrifugo.")
	b.Centrifuge.BatchPublish(ctx, messages)
}

func GetChartsChannels() map[string]map[string]*domain.ChartChannel {
	m := domain.GetAvailableMarkets()
	c := make(map[string]map[string]*domain.ChartChannel, len(m))
	for _, market := range m {
		resolutions := domain.GetAvailableResolutions()
		marketChannels := make(
			map[string]*domain.ChartChannel,
			len(resolutions),
		)
		for _, resolution := range resolutions {
			marketChannels[resolution] = NewChartChannel(market, resolution)
		}
		c[market] = marketChannels
	}
	return c
}

func NewChartChannel(market string, resolution string) *domain.ChartChannel {
	name := fmt.Sprintf(
		"%s_%s_%s",
		domain.CandleChartChannelPrefix,
		market,
		resolution,
	)
	return &domain.ChartChannel{
		Name:       name,
		Market:     market,
		Resolution: resolution,
	}
}
