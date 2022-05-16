package candle

import (
	"context"
	"strconv"
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

// CronCandleGenerationStart generates candle for websocket pushing every min
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
	resolution string,
) (*domain.Chart, error) {
	from := time.Unix(
		s.Aggregator.GetCurrentResolutionStartTimestamp(resolution),
		0,
	)
	to := time.Now()

	chart := s.GetCandleByResolution(ctx, market, resolution, from, to)

	if chart != nil {
		chart.SetMarket(market)
		chart.SetResolution(resolution)
	}

	return chart, nil
}

func (s Service) GetCandleByResolution(ctx context.Context, market string, resolution string, from time.Time, to time.Time) *domain.Chart {
	logger.FromContext(ctx).WithField(
		"resolution",
		resolution,
	).WithField(
		"from",
		from,
	).WithField(
		"to",
		to,
	).Infof("[CandleService] Call GetCandleByResolution method.")
	var chart *domain.Chart
	switch resolution {
	case domain.Candle1MResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.MinuteUnit, 1, from, to)
	case domain.Candle3MResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.MinuteUnit, 3, from, to)
	case domain.Candle5MResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.MinuteUnit, 5, from, to)
	case domain.Candle15MResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.MinuteUnit, 15, from, to)
	case domain.Candle30MResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.MinuteUnit, 30, from, to)
	case domain.Candle1HResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.HourUnit, 1, from, to)
	case domain.Candle2HResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.HourUnit, 2, from, to)
	case domain.Candle4HResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.HourUnit, 4, from, to)
	case domain.Candle6HResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.HourUnit, 6, from, to)
	case domain.Candle12HResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.HourUnit, 12, from, to)
	case domain.Candle1DResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.DayUnit, 1, from, to)
	case domain.Candle1MHResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.MonthUnit, 1, from, to)
	default:
		logger.FromContext(context.Background()).WithField(
			"resolution",
			resolution,
		).Errorf("Unsupported resolution.")

		println(chart.H)
		return &domain.Chart{}
	}

	if chart != nil {
		chart.SetResolution(resolution)
	}

	return chart
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
				s.PushUpdatedCurrentCharts(ctx, deal.Data.Market)
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

	if len(chts) == 0 {
		return
	}

	s.eventsBroker.Publish(domain.EvTypeCharts, domain.NewEvent(ctx, chts))
}

func (s *Service) GetChart(
	ctx context.Context,
	market string,
	resolution string,
	from time.Time,
	to time.Time,
) (domain.ChartResponse, interface{}) {
	chart := s.GetCandleByResolution(ctx, market, resolution, from, to)
	return s.makeChartResponse(chart), nil
}

func (s *Service) makeChartResponse(chart *domain.Chart) domain.ChartResponse {
	r := domain.ChartResponse{
		Symbol: chart.Symbol,
		H:      make([]string, len(chart.H)),
		L:      make([]string, len(chart.H)),
		O:      make([]string, len(chart.H)),
		C:      make([]string, len(chart.H)),
		V:      make([]string, len(chart.H)),
		T:      make([]string, len(chart.H)),
	}
	//  convert chart values primitives decimal128 to string
	for i := 0; i < len(chart.V); i++ {
		// convert decimal128 to string
		r.O[i] = chart.O[i].String()
		r.H[i] = chart.H[i].String()
		r.L[i] = chart.L[i].String()
		r.C[i] = chart.C[i].String()
		r.V[i] = chart.V[i].String()
		r.T[i] = strconv.FormatInt(chart.T[i], 10)
	}

	return r
}
