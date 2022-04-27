package candle

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"bitbucket.org/novatechnologies/common/infra/logger"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"bitbucket.org/novatechnologies/ohlcv/domain"
)

const chartsPublishingTimeout = 16 * time.Second

type Service struct {
	DealsDbCollection  *mongo.Collection
	Markets            map[string]string
	AvailableIntervals []string
	marketDataBus      domain.EventManager
}

func NewService(
	dealsDbCollection *mongo.Collection,
	markets map[string]string,
	availableIntervals []string,
	internalBus domain.EventManager,
) *Service {
	s := &Service{
		DealsDbCollection:  dealsDbCollection,
		Markets:            markets,
		AvailableIntervals: availableIntervals,
		marketDataBus:      internalBus,
	}

	go s.subForDealsToPubCharts()

	return s
}

func (s *Service) subForDealsToPubCharts() {
	s.marketDataBus.Subscribe(
		domain.ETypeTrades, func(dealMsg *domain.Event) error {
			deal := dealMsg.MustGetDeal()

			ctx, cancel := context.WithTimeout(
				context.Background(),
				chartsPublishingTimeout,
			)
			defer cancel()

			for _, interval := range s.AvailableIntervals {
				chart, err := s.GetLatestChart(ctx, deal.Market, interval)
				if err != nil {
					logger.FromContext(context.Background()).
						WithField("interval", interval).
						WithField("market", deal.Market).
						Errorf(
							"CandleService.GetLatestChart method error: %v.",
							err,
						)

					continue
				}
				//s.UpdatedCandles <- upd
				s.marketDataBus.Publish(
					domain.ETypeCharts,
					domain.NewEvent(ctx, *chart),
				)
			}

			return nil
		},
	)
}

// CronCandleGenerationStart triggers pushing into websocket a new candle every
// minute (example: empty candles)
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
					// s.PushUpdatedCandleEvent(ctx, market)
				}
			}
		}
	}()
}

func (s Service) GetMinuteCandles(
	ctx context.Context,
	market string,
	period ...time.Time,
) ([]*domain.Candle, error) {
	logger.FromContext(ctx).WithField(
		"market",
		market,
	).Infof("[CandleService] Call GetMinuteCandles")
	matchStage := bson.D{
		{"$match", bson.D{
			{"market", market},
		},
		}}

	/*if len(period) == 2 {
		from, to := period[0], period[1]
		matchStage[0].Value = append(
			matchStage[0].Value.(bson.D),
			bson.E{KeyInPEM: "time", Value: bson.D{
				{"$gte", primitive.NewDateTimeFromTime(from)},
			}},
			bson.E{KeyInPEM: "time", Value: bson.D{
				{"$lte", primitive.NewDateTimeFromTime(to)},
			}},
		)
	}*/

	groupStage := bson.D{{"$group", bson.D{
		{"_id", bson.D{
			{"market", "$market"},
			{"time", bson.D{
				{" $dateTrunc", bson.D{
					{"date", "$time"},
					{"unit", "$minute"},
					{"binSize", 1},
				}},
			}},
		}},
		{"high", bson.D{{"$max", "$price"}}},
		{"low", bson.D{{"$min", "$price"}}},
		{"open", bson.D{{"$first", "$price"}}},
		{"close", bson.D{{"$last", "$price"}}},
		{"volume", bson.D{{"$sum", "$volume"}}},
		{"timestamp", bson.D{{"$first", "$time"}}},
	}}}

	sortStage := bson.D{{"$sort", bson.D{
		{
			"_id.time", 1,
		},
	}}}

	opts := options.Aggregate()
	adu := true
	opts.AllowDiskUse = &adu
	cursor, err := s.DealsDbCollection.Aggregate(
		ctx,
		mongo.Pipeline{matchStage, groupStage, sortStage},
		opts,
	)

	if err != nil {
		logger.FromContext(ctx).WithField(
			"error",
			err,
		).Errorf("[CandleService]Failed apply a aggregation function on the collection.")
		return nil, err
	}

	candles := make([]*domain.Candle, 0)

	err = cursor.All(ctx, &candles)

	if err != nil {
		logger.FromContext(ctx).WithField(
			"error",
			err,
		).Errorf("[CandleService]Failed apply a aggregation function on the collection.")
		return nil, err
	}

	if len(candles) == 0 {
		logger.FromContext(ctx).WithField(
			"candleCount",
			len(candles),
		).WithField("err", err).WithField(
			"err",
			period,
		).Infof("Candles not found.")
		return nil, err
	}
	logger.FromContext(ctx).WithField(
		"candleCount",
		len(candles),
	).Infof("Success get candles.")

	return candles, err
}

func (s Service) GetLatestChart(
	ctx context.Context,
	market string,
	interval string,
) (*domain.Chart, error) {
	cs, err := s.GetMinuteCandles(ctx, market)
	chart := s.AggregateCandleToChartByInterval(cs, market, interval, 1)
	chart.SetMarket(market)
	chart.SetInterval(interval)

	return chart, err
}

func (s Service) AggregateCandleToChartByInterval(
	candles []*domain.Candle,
	market string,
	interval string,
	count int,
) *domain.Chart {
	var chart *domain.Chart

	logger.FromContext(context.Background()).WithField(
		"interval",
		interval,
	).Infof("[CandleService] Call AggregateCandleToChartByInterval method.")
	switch interval {
	case domain.Candle1MInterval:
		chart = s.aggregateMinCandlesToChart(candles, market, 1, count)
	case domain.Candle3MInterval:
		chart = s.aggregateMinCandlesToChart(candles, market, 3, count)
	case domain.Candle5MInterval:
		chart = s.aggregateMinCandlesToChart(candles, market, 5, count)
	case domain.Candle15MInterval:
		chart = s.aggregateMinCandlesToChart(candles, market, 15, count)
	case domain.Candle30MInterval:
		chart = s.aggregateMinCandlesToChart(candles, market, 30, count)
	case domain.Candle1HInterval:
		chart = s.aggregateHoursCandlesToChart(candles, market, 1, count)
	case domain.Candle2HInterval:
		chart = s.aggregateHoursCandlesToChart(candles, market, 2, count)
	case domain.Candle4HInterval:
		chart = s.aggregateHoursCandlesToChart(candles, market, 4, count)
	case domain.Candle6HInterval:
		chart = s.aggregateHoursCandlesToChart(candles, market, 6, count)
	case domain.Candle12HInterval:
		chart = s.aggregateHoursCandlesToChart(candles, market, 12, count)
	case domain.Candle1DInterval:
		chart = s.aggregateHoursCandlesToChart(candles, market, 24, count)
	case domain.Candle1MHInterval:
		chart = s.aggregateMonthCandlesToChart(candles, market, count)
	default:
		logger.FromContext(context.Background()).WithField(
			"interval",
			interval,
		).Errorf("Unsupported interval.")
	}

	return chart
}

func (s Service) aggregateMinCandlesToChart(
	candles []*domain.Candle,
	_ string,
	minute int,
	_ int,
) *domain.Chart {
	result := make(map[int64]*domain.Candle)

	var min int
	var mod int
	var mul time.Duration
	var timestamp int64
	now := time.Now()
	currentTs := now.Add(time.Duration(now.Minute()%minute) * -time.Minute).Unix()
	for _, candle := range candles {
		var comparedCandle *domain.Candle
		min = int(int64(candle.Timestamp.Minute()))
		mod = min % minute
		mul = time.Duration(mod) * -time.Minute
		timestamp = candle.Timestamp.Add(mul).Unix()
		c := result[timestamp]

		if c != nil {
			comparedCandle = s.compare(c, candle)
		} else {
			comparedCandle = candle
		}

		result[timestamp] = comparedCandle
		if currentTs == timestamp {
			result[currentTs] = comparedCandle
		}
	}

	chart := s.GenerateChart(result)

	return chart
}

func (s Service) compare(
	c *domain.Candle,
	candle *domain.Candle,
) *domain.Candle {
	comparedCandle := &domain.Candle{}
	if c.Timestamp.Unix() < candle.Timestamp.Unix() {
		comparedCandle.Open = c.Open
		comparedCandle.Close = candle.Close
	} else {
		comparedCandle.Open = candle.Open
		comparedCandle.Close = c.Close
	}

	cHight, _ := CompareDecimal128(c.High, candle.High)
	if cHight == -1 {
		comparedCandle.High = candle.High
	}
	cLow, _ := CompareDecimal128(c.Low, candle.Low)
	if cLow == 1 {
		comparedCandle.Low = candle.Low
	}
	dv1, _ := decimal.NewFromString(c.Volume.String())
	dv2, _ := decimal.NewFromString(candle.Volume.String())
	resultVolume, _ := primitive.ParseDecimal128(dv1.Add(dv2).String())
	comparedCandle.Volume = resultVolume
	comparedCandle.Timestamp = candle.Timestamp

	return comparedCandle
}

func (s *Service) aggregateHoursCandlesToChart(
	candles []*domain.Candle,
	_ string,
	hour int,
	_ int,
) *domain.Chart {
	result := make(map[int64]*domain.Candle)

	var min int
	var mod int
	var mul time.Duration
	var timestamp int64
	for _, candle := range candles {
		min = int(int64(candle.Timestamp.Hour()))
		mod = min % hour
		mul = time.Duration(mod) * -time.Hour
		timestamp = candle.Timestamp.Add(mul).Unix()
		c := result[timestamp]
		if c != nil {
			result[timestamp] = s.compare(c, candle)
		} else {
			result[timestamp] = candle
		}
	}

	chart := s.GenerateChart(result)

	return chart
}

func (s *Service) aggregateMonthCandlesToChart(
	candles []*domain.Candle,
	_ string,
	_ int,
) *domain.Chart {
	result := make(map[int64]*domain.Candle)

	var timestamp int64
	for _, candle := range candles {
		timestamp = time.Date(
			candle.Timestamp.Year(),
			candle.Timestamp.Month(),
			1,
			0,
			0,
			0,
			0,
			time.Local,
		).Unix()
		c := result[timestamp]
		if c != nil {
			result[timestamp] = s.compare(c, candle)
		} else {
			result[timestamp] = candle
		}
	}

	chart := s.GenerateChart(result)

	return chart
}

func (s *Service) GenerateChart(result map[int64]*domain.Candle) *domain.Chart {
	chart := &domain.Chart{
		O: make([]string, 0),
		H: make([]string, 0),
		L: make([]string, 0),
		C: make([]string, 0),
		V: make([]string, 0),
		T: make([]int64, 0),
	}

	for t, aggregatedCandle := range result {
		chart.O = append(chart.O, aggregatedCandle.Open.String())
		chart.H = append(chart.H, aggregatedCandle.High.String())
		chart.L = append(chart.L, aggregatedCandle.Low.String())
		chart.C = append(chart.C, aggregatedCandle.Close.String())
		chart.V = append(chart.V, aggregatedCandle.Volume.String())
		chart.T = append(chart.T, t)
	}

	return chart
}

func CompareDecimal128(d1, d2 primitive.Decimal128) (int, error) {
	b1, exp1, err := d1.BigInt()
	if err != nil {
		return 0, err
	}
	b2, exp2, err := d2.BigInt()
	if err != nil {
		return 0, err
	}

	sign := b1.Sign()
	if sign != b2.Sign() {
		if b1.Sign() > 0 {
			return 1, nil
		} else {
			return -1, nil
		}
	}

	if exp1 == exp2 {
		return b1.Cmp(b2), nil
	}

	if sign < 0 {
		if exp1 < exp2 {
			return 1, nil
		}
		return -1, nil
	} else {
		if exp1 < exp2 {
			return -1, nil
		}

		return 1, nil
	}
}
