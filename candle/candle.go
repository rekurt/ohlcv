package candle

import (
	"context"
	"time"

	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/domain"
)

type Service struct {
	Storage      *Storage
	Aggregator   *Aggregator
	broadcaster  domain.Broadcaster
	eventsBroker domain.EventsBroker
}

func NewService(storage *Storage, aggregator *Aggregator, internalBus domain.EventsBroker) *Service {
	return &Service{
		Storage:      storage,
		Aggregator:   aggregator,
		eventsBroker: internalBus,
	}
}

func (s Service) GetCurrentCandle(
	ctx context.Context,
	market string,
	resolution string,
) (*domain.Chart, error) {
	from := time.Unix(
		s.Aggregator.GetResolutionStartTimestampByTime(resolution, time.Now()),
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
	).Tracef("[CandleService] Call GetCandleByResolution method.")
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
	case domain.Candle1HResolution,
		domain.Candle1H2Resolution:
		chart = s.Storage.GetCandles(ctx, market, domain.HourUnit, 1, from, to)
	case domain.Candle2HResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.HourUnit, 2, from, to)
	case domain.Candle2H2Resolution:
		chart = s.Storage.GetCandles(ctx, market, domain.HourUnit, 2, from, to)
	case domain.Candle4HResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.HourUnit, 4, from, to)
	case domain.Candle4H2Resolution:
		chart = s.Storage.GetCandles(ctx, market, domain.HourUnit, 4, from, to)
	case domain.Candle6HResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.HourUnit, 6, from, to)
	case domain.Candle6H2Resolution:
		chart = s.Storage.GetCandles(ctx, market, domain.HourUnit, 6, from, to)
	case domain.Candle12HResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.HourUnit, 12, from, to)
	case domain.Candle12H2Resolution:
		chart = s.Storage.GetCandles(ctx, market, domain.HourUnit, 12, from, to)
	case domain.Candle1DResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.DayUnit, 1, from, to)
	case domain.Candle1MHResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.MonthUnit, 1, from, to)
	case domain.Candle1MH2Resolution:
		chart = s.Storage.GetCandles(ctx, market, domain.MonthUnit, 1, from, to)
	case domain.Candle1WResolution:
		chart = s.Storage.GetCandles(ctx, market, domain.WeekUnit, 1, from, to)
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

func (s *Service) GetChart(
	ctx context.Context,
	market string,
	resolution string,
	from time.Time,
	to time.Time,
) (domain.ChartResponse, interface{}) {
	chart := s.GetCandleByResolution(ctx, market, resolution, from, to)
	return domain.MakeChartResponse(market, chart), nil
}
