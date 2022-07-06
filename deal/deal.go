package deal

import (
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/client/market"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	pubsub "bitbucket.org/novatechnologies/common/events"
	"bitbucket.org/novatechnologies/common/events/topics"
	"bitbucket.org/novatechnologies/common/infra/logger"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"bitbucket.org/novatechnologies/interfaces/matcher"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"bitbucket.org/novatechnologies/ohlcv/domain"
)

func TopicName(prefix string) string {
	return prefix + "_" + topics.MatcherMDDeals
}

type Service struct {
	DbCollection *mongo.Collection
	Markets      map[string]string
	marketsInfo  []market.Market
}

func NewService(dbCollection *mongo.Collection, markets map[string]string, marketsInfo []market.Market) *Service {
	return &Service{
		DbCollection: dbCollection,
		Markets:      markets,
		marketsInfo:  marketsInfo,
	}
}

func (s *Service) SaveDeal(
	ctx context.Context,
	dealMessage *matcher.Deal,
) (*domain.Deal, error) {
	defer func() {
		if r := recover(); r != "" {
			logger.FromContext(ctx).Errorf(r)
			// TODO: sending notification manually to the sentry or alternative.
		}
	}()

	if dealMessage.TakerOrderId == "" || dealMessage.MakerOrderId == "" {
		logger.FromContext(ctx).Infof("The deal have empty TakerOrderId or MakerOrderId field. Skip. Dont save to mongo.")
		return nil, nil
	}
	marketName := s.Markets[dealMessage.Market]
	deal := &domain.Deal{
		T: primitive.NewDateTimeFromTime(time.Unix(0, dealMessage.CreatedAt)),
		Data: domain.DealData{
			Price:        domain.MustParseDecimal(dealMessage.Price),
			Volume:       domain.MustParseDecimal(dealMessage.Amount),
			DealId:       dealMessage.Id,
			Market:       marketName,
			IsBuyerMaker: dealMessage.IsBuyerMaker,
		},
	}
	if err := deal.Validate(); err != nil {
		return nil, err
	}

	_, err := s.DbCollection.InsertOne(ctx, deal)
	if err != nil {
		logger.FromContext(ctx).WithField(
			"error",
			err.Error(),
		).Errorf("[DealService]Failed save deal.", deal)
		return nil, err
	}
	return deal, nil
}

func (s *Service) GetLastTrades(
	ctx context.Context,
	symbol string,
	limit int32,
) ([]domain.Deal, error) {
	ctx, cancelFunc := context.WithTimeout(ctx, time.Second)
	defer cancelFunc()

	if strings.TrimSpace(symbol) == "" || limit <= 0 || limit >= 1000 {
		logger.FromContext(ctx).Infof(
			"Incorrect args: symbol='%s', limit=%d",
			symbol,
			limit,
		)
		return nil, nil
	}
	cursor, err := s.DbCollection.Find(
		ctx,
		bson.M{"data.market": symbol},
		options.Find().SetLimit(int64(limit)),
	)
	if err != nil {
		logger.FromContext(ctx).WithField(
			"error",
			err.Error(),
		).Errorf("[DealService]Failed GetLastTrades")
		return nil, err
	}
	var deals []domain.Deal
	err = cursor.All(ctx, &deals)
	if err != nil {
		logger.FromContext(ctx).WithField(
			"error",
			err.Error(),
		).Errorf("[DealService]Failed GetLastTrades")
		return nil, err
	}
	return deals, nil
}

func (s *Service) GetTickerPriceChangeStatistics(ctx context.Context, duration time.Duration, market string) ([]domain.TickerPriceChangeStatistics, error) {
	matchStageValue := bson.D{
		{"t", bson.D{
			{"$gte", primitive.NewDateTimeFromTime(time.Now().Add(-duration))},
		}},
	}
	if strings.TrimSpace(market) != "" {
		matchStageValue = append(matchStageValue, bson.E{Key: "data.market", Value: market})
	}
	sortStage := bson.D{{"$sort", bson.D{
		{
			"t", 1,
		},
	}}}
	groupStage := bson.D{
		{"$group",
			bson.D{
				{"_id", "$data.market"},
				{"volume", bson.D{{"$sum", "$data.volume"}}},
				{"quoteVolume", bson.D{{"$sum", bson.D{{"$multiply", bson.A{"$data.price", "$data.volume"}}}}}},
				{"count", bson.D{{"$count", bson.M{}}}},
				{"highPrice", bson.D{{"$max", "$data.price"}}},
				{"lowPrice", bson.D{{"$min", "$data.price"}}},
				{"openPrice", bson.D{{"$first", "$data.price"}}},
				{"closePrice", bson.D{{"$last", "$data.price"}}},
				{"openTime", bson.D{{"$first", "$t"}}},
				{"closeTime", bson.D{{"$last", "$t"}}},
				{"firstId", bson.D{{"$first", "$data.dealid"}}},
				{"lastId", bson.D{{"$last", "$data.dealid"}}},
				{"lastQty", bson.D{{"$last", "$data.volume"}}},
			},
		},
	}
	/*
		{
		  "t": {
		    "$date": {
		      "$numberLong": "1655306454358"
		    }
		  },
		  "data": {
		    "dealid": "ffb51c65-8d59-453f-a172-9f71f4293516",
		    "isbuyermaker": false,
		    "market": "USDT_TRX",
		    "price": {
		      "$numberDecimal": "0.056000"
		    },
		    "volume": {
		      "$numberDecimal": "113.60000000"
		    }
		  },
		  "_id": {
		    "$oid": "62a9f8d7fdca7f38bd70146d"
		  }
		}

				db.collection.aggregate({
			  $group : {
			     _id : 'weighted average', // build any group key ypo need
			     quoteVolume: { $sum: { $multiply: [ "$price", "$quantity" ] } },
			     denominator: { $sum: "$quantity" }
			  }
			}, {
			  $project: {
			    average: { $divide: [ "$quoteVolume", "$denominator" ] }
			  }
			}

			  "x": "0.0009",      // First trade(F)-1 price (first trade before the 24hr rolling window)
			  "b": "0.0024",      // Best bid price
			  "B": "10",          // Best bid quantity
			  "a": "0.0026",      // Best ask price
			  "A": "100",         // Best ask quantity
	*/
	aggregateOptions := options.Aggregate()
	aggregateOptions.SetAllowDiskUse(true)
	deadline, ok := ctx.Deadline()
	if ok {
		aggregateOptions.SetMaxTime(deadline.Sub(time.Now()))
	}
	aggregate, err := s.DbCollection.Aggregate(
		ctx,
		mongo.Pipeline{bson.D{{Key: "$match", Value: matchStageValue}}, sortStage, groupStage},
		aggregateOptions,
	)
	if err != nil {
		return nil, fmt.Errorf("GetTickerPriceChangeStatistics: Aggregate error '%w'", err)
	}
	var resp []bson.M
	if err = aggregate.All(ctx, &resp); err != nil {
		return nil, fmt.Errorf("GetTickerPriceChangeStatistics: aggregate.All error '%w'", err)
	}
	if len(resp) == 0 {
		return nil, nil
	}
	statistics := make([]domain.TickerPriceChangeStatistics, 0, len(resp))
	for _, v := range resp {
		statistics = append(statistics, parseStatistics(v))
	}
	return statistics, nil
}

func parseStatistics(m bson.M) domain.TickerPriceChangeStatistics {
	closePrice := m["closePrice"].(primitive.Decimal128)
	openPrice := m["openPrice"].(primitive.Decimal128)
	quoteVolume := m["quoteVolume"].(primitive.Decimal128)
	volume := m["volume"].(primitive.Decimal128)
	priceChange, priceChangePercent := calcChange(closePrice, openPrice)

	return domain.TickerPriceChangeStatistics{
		Symbol:             m["_id"].(string),
		WeightedAvgPrice:   calcVwap(quoteVolume, volume),
		LastPrice:          closePrice.String(),
		OpenPrice:          openPrice.String(),
		HighPrice:          m["highPrice"].(primitive.Decimal128).String(),
		LowPrice:           m["lowPrice"].(primitive.Decimal128).String(),
		Volume:             volume.String(),
		QuoteVolume:        quoteVolume.String(),
		OpenTime:           m["openTime"].(primitive.DateTime).Time().UnixMilli(),
		CloseTime:          m["closeTime"].(primitive.DateTime).Time().UnixMilli(),
		FirstId:            m["firstId"].(string),
		LastId:             m["lastId"].(string),
		LastQty:            m["lastQty"].(primitive.Decimal128).String(),
		Count:              int(m["count"].(int32)),
		PriceChange:        strconv.FormatFloat(priceChange, 'f', 8, 64),
		PriceChangePercent: strconv.FormatFloat(priceChangePercent, 'f', 8, 64),
	}
}

func calcVwap(quoteVolume, volume primitive.Decimal128) string {
	quoteVolumeF, err := strconv.ParseFloat(quoteVolume.String(), 64)
	if err != nil {
		return ""
	}
	volumeF, err := strconv.ParseFloat(volume.String(), 64)
	if err != nil {
		return ""
	}
	vwap := quoteVolumeF / volumeF
	return strconv.FormatFloat(vwap, 'f', 4, 64)
}

func calcChange(closePrice, openPrice primitive.Decimal128) (float64, float64) {
	closePriceF, err := strconv.ParseFloat(closePrice.String(), 64)
	if err != nil {
		return 0, 0
	}
	openPriceF, err := strconv.ParseFloat(openPrice.String(), 64)
	if err != nil {
		return 0, 0
	}
	change := closePriceF - openPriceF
	priceChangePercent := change / openPriceF
	return change, priceChangePercent
}

func (s *Service) RunConsuming(ctx context.Context, consumer pubsub.Subscriber, topic string, currentCandles candle.CurrentCandles) {
	go func() {
		err := func() error {
			return consumer.Consume(
				ctx,
				topic,
				func(
					ctx context.Context,
					metadata map[string]string,
					msg []byte,
				) error {
					dealMessage := matcher.Deal{}
					if err := proto.Unmarshal(msg, &dealMessage); err != nil {
						logger.FromContext(ctx).
							WithField("method", "consumer.deals.Unmarshal").
							Errorf(err)

						return errors.Wrap(
							err,
							"unmarshal error with protobuf deals msg",
						)
					}
					err := currentCandles.AddDeal(dealMessage)
					if err != nil {
						logger.FromContext(ctx).
							WithField("method", "currentCandles.AddDeal in consuming").
							Errorf(err)
					}
					if deal, err := s.SaveDeal(ctx, &dealMessage); err != nil {
						return errors.Wrapf(err, "while saving deal %v into DB", deal)
					}
					return nil
				},
			)
		}()
		if err != nil {
			logger.FromContext(ctx).
				WithField("err", err).
				WithField("svc", "DealsService").
				Errorf("Consuming session was finished with error", err)
		}
	}()
}

func (s *Service) GetAvgPrice(ctx context.Context, duration time.Duration, market string) (string, error) {
	matchStageValue := bson.D{
		{"t", bson.D{
			{"$gte", primitive.NewDateTimeFromTime(time.Now().Add(-duration))},
		}},
		bson.E{Key: "data.market", Value: market},
	}
	if strings.TrimSpace(market) == "" {
		return "0", fmt.Errorf("can't GetAvgPrice, empty symbol")
	}
	groupStage := bson.D{
		{"$group",
			bson.D{
				{"_id", "$data.market"},
				{"avg", bson.D{{"$avg", "$data.price"}}},
			},
		},
	}
	aggregateOptions := options.Aggregate()
	deadline, ok := ctx.Deadline()
	if ok {
		aggregateOptions.SetMaxTime(deadline.Sub(time.Now()))
	}
	aggregate, err := s.DbCollection.Aggregate(
		ctx,
		mongo.Pipeline{bson.D{{Key: "$match", Value: matchStageValue}}, groupStage},
		aggregateOptions,
	)
	if err != nil {
		return "0", fmt.Errorf("GetAvgPrice: Aggregate error '%w'", err)
	}
	var resp []bson.M
	if err = aggregate.All(ctx, &resp); err != nil {
		return "0", fmt.Errorf("GetAvgPrice: aggregate.All error '%w'", err)
	}
	if len(resp) == 0 {
		return "0", nil
	}
	return s.roundByMarket(resp[0]["avg"].(primitive.Decimal128), market)
}

func (s *Service) roundByMarket(decimal128 primitive.Decimal128, market string) (string, error) {
	f, err := strconv.ParseFloat(decimal128.String(), 64)
	if err != nil {
		return "0", nil
	}
	return strconv.FormatFloat(
		f,
		'f',
		findPrec(market, s.marketsInfo),
		64), nil
}

func findPrec(market string, info []market.Market) int {
	for _, i := range info {
		if i.Name == market {
			return int(i.BasePrecision)
		}
	}
	return 4
}
