package candle

import (
	"context"
	"time"

	"bitbucket.org/novatechnologies/common/infra/logger"

	"bitbucket.org/novatechnologies/ohlcv/domain"
)

const chartsPubTimeout = 16 * time.Second

type Service struct {
	Storage              *Storage
	Aggregator           *Agregator
	Markets              map[string]string
	AvailableResolutions []string
	broadcaster          domain.Broadcaster
	eventsBroker         domain.EventsBroker
}

func NewService(
	storage *Storage,
	aggregator *Agregator,
	markets map[string]string,
	availableResolutions []string,
	internalBus domain.EventsBroker,
) *Service {
	return &Service{
		Storage:              storage,
		Aggregator:           aggregator,
		Markets:              markets,
		AvailableResolutions: availableResolutions,
		eventsBroker:         internalBus,
	}
}

//CronCandleGenerationStart generates candle for websocket pushing every min
// (example: empty candles).
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
	from := time.Unix(
		s.Aggregator.GetCurrentResolutionStartTimestamp(resolutions),
		0,
	)
	to := time.Now()
	cs, err := s.Storage.GetMinuteCandles(ctx, market, from, to)
	chart := s.Aggregator.AggregateCandleToChartByResolution(
		cs,
		market,
		resolutions,
		1,
	)
	chart.SetMarket(market)
	chart.SetResolution(resolutions)

	return chart, err
}

// SubscribeForDeals subscribes for new deals to update and publish particular
// charts.
func (s *Service) SubscribeForDeals() {
	go s.eventsBroker.Subscribe(
		domain.EvTypeDeals,
		func(dealEvent *domain.Event) error {
			deals := dealEvent.MustGetDeals()

			ctx, cancel := context.WithTimeout(dealEvent.Ctx, chartsPubTimeout)
			defer cancel()

			for _, deal := range deals {
				s.PushUpdatedCurrentCharts(ctx, deal.Market)
			}

			return nil
		},
	)
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

	s.eventsBroker.Publish(domain.EvTypeCharts, domain.NewEvent(ctx, chts))
}

func (s *Service) GetChart(
	ctx context.Context,
	market string,
	resolution string,
	from time.Time,
	to time.Time,
) (interface{}, interface{}) {
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
