package candle

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"context"
	"time"
)

type Service struct {
	Storage              *Storage
	Aggregator           *Agregator
	broadcaster          domain.Broadcaster
	Markets              map[string]string
	AvailableResolutions []string
}

func NewService(s *Storage, a *Agregator, b domain.Broadcaster, markets map[string]string, availableResolution []string) *Service {
	return &Service{Storage: s, Aggregator: a, broadcaster: b, Markets: markets, AvailableResolutions: availableResolution}
}

//Generation for websocket pushing new candle every min (example: empty candles)
func (s *Service) CronCandleGenerationStart(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		done := make(chan bool)
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				for _, market := range s.Markets {
					logger.FromContext(ctx).WithField(
						"market",
						market,
					).Infof("[CronCandleGenerationStart]Getting new candle for the market.")
					s.PushUpdatedCurrentCharts(ctx, market)
				}
			}
		}
	}()
}

func (s Service) GetCurrentCandle(
	ctx context.Context,
	market string,
	resolutions string,
) (*domain.Chart, error) {
	from := time.Unix(s.Aggregator.GetCurrentResolutionStartTimestamp(resolutions), 0)
	to := time.Now()
	cs, err := s.Storage.GetMinuteCandles(ctx, market, from, to)
	chart := s.Aggregator.AggregateCandleToChartByResolution(cs, market, resolutions, 1)
	chart.SetMarket(market)
	chart.SetResolution(resolutions)

	return chart, err
}

func (s *Service) PushUpdatedCurrentCharts(ctx context.Context, market string) {
	chts := make([]*domain.Chart, 0)
	for _, resolution := range s.AvailableResolutions {
		logger.FromContext(context.Background()).
			WithField("resolution", resolution).
			WithField("market", market).
			Infof("[CandleService] Call PushLastUpdatedCandle method.")
		upd, _ := s.GetCurrentCandle(ctx, market, resolution)
		if upd != nil {
			chts = append(chts, upd)
		}
	}

	s.broadcaster.BroadcastCandleCharts(ctx, chts)
}

func (s *Service) GetChart(ctx context.Context, market string, resolution string, from time.Time, to time.Time) (interface{}, interface{}) {
	candles, err := s.Storage.GetMinuteCandles(ctx, market, from, to)
	if err != nil {
		logger.FromContext(ctx).
			WithField("err", err).
			WithField("market", market).
			WithField("resolution", resolution).
			Errorf("Cannot get the chart.")
		return &domain.Chart{}, err
	}

	chart := s.Aggregator.AggregateCandleToChartByResolution(
		candles, market, resolution, 0,
	)

	return chart, nil
}
