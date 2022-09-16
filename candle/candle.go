package candle

import (
	"bitbucket.org/novatechnologies/ohlcv/internal/model"
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
	resolution model.Resolution,
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

func (s Service) GetCandleByResolution(ctx context.Context, market string, resolution model.Resolution, from time.Time, to time.Time) *domain.Chart {
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
	case model.Candle1MResolution:
		chart = s.Storage.GetCandles(ctx, market, model.MinuteUnit, 1, from, to)
	case model.Candle3MResolution:
		chart = s.Storage.GetCandles(ctx, market, model.MinuteUnit, 3, from, to)
	case model.Candle5MResolution:
		chart = s.Storage.GetCandles(ctx, market, model.MinuteUnit, 5, from, to)
	case model.Candle15MResolution:
		chart = s.Storage.GetCandles(ctx, market, model.MinuteUnit, 15, from, to)
	case model.Candle30MResolution:
		chart = s.Storage.GetCandles(ctx, market, model.MinuteUnit, 30, from, to)
	case model.Candle1HResolution,
		model.Candle1H2Resolution:
		chart = s.Storage.GetCandles(ctx, market, model.HourUnit, 1, from, to)
	case model.Candle2HResolution:
		chart = s.Storage.GetCandles(ctx, market, model.HourUnit, 2, from, to)
	case model.Candle2H2Resolution:
		chart = s.Storage.GetCandles(ctx, market, model.HourUnit, 2, from, to)
	case model.Candle4HResolution:
		chart = s.Storage.GetCandles(ctx, market, model.HourUnit, 4, from, to)
	case model.Candle4H2Resolution:
		chart = s.Storage.GetCandles(ctx, market, model.HourUnit, 4, from, to)
	case model.Candle6HResolution:
		chart = s.Storage.GetCandles(ctx, market, model.HourUnit, 6, from, to)
	case model.Candle6H2Resolution:
		chart = s.Storage.GetCandles(ctx, market, model.HourUnit, 6, from, to)
	case model.Candle12HResolution:
		chart = s.Storage.GetCandles(ctx, market, model.HourUnit, 12, from, to)
	case model.Candle12H2Resolution:
		chart = s.Storage.GetCandles(ctx, market, model.HourUnit, 12, from, to)
	case model.Candle1DResolution:
		chart = s.Storage.GetCandles(ctx, market, model.DayUnit, 1, from, to)
	case model.Candle1MHResolution:
		chart = s.Storage.GetCandles(ctx, market, model.MonthUnit, 1, from, to)
	case model.Candle1MH2Resolution:
		chart = s.Storage.GetCandles(ctx, market, model.MonthUnit, 1, from, to)
	case model.Candle1WResolution:
		chart = s.Storage.GetCandles(ctx, market, model.WeekUnit, 1, from, to)
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
	resolution model.Resolution,
	from time.Time,
	to time.Time,
) domain.ChartResponse {
	chart := s.GetCandleByResolution(ctx, market, resolution, from, to)
	return domain.MakeChartResponse(market, chart)
}
