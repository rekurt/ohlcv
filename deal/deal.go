package deal

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/novatechnologies/ohlcv/client/market"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"bitbucket.org/novatechnologies/common/events/topics"
	"bitbucket.org/novatechnologies/common/infra/logger"

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
	t := time.Unix(0, dealMessage.CreatedAt)
	marketName := s.Markets[dealMessage.Market]
	deal := &domain.Deal{
		T: primitive.NewDateTimeFromTime(t),
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
	var deals = make([]*domain.Deal, 1)
	deals[0] = deal

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
	fromTime := primitive.NewDateTimeFromTime(time.Now().Add(-duration))
	matchStageValue := bson.D{
		{"t", bson.D{
			{"$gte", fromTime},
		}},
		{"data.market", market},
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
	lookupStage := bson.D{
		{"$lookup",
			bson.D{
				{"from", s.DbCollection.Name()},
				{"localField", "_id"},
				{"foreignField", "data.market"},
				{"pipeline",
					bson.A{
						bson.D{{"$match", bson.D{{"t", bson.D{{"$lt", fromTime}}}}}},
						bson.D{{"$sort", bson.D{{"t", 1}}}},
						bson.D{{"$limit", 1}},
					},
				},
				{"as", "prev_window_trade"},
			},
		},
	}
	aggregateOptions := options.Aggregate()
	aggregateOptions.SetAllowDiskUse(true)
	deadline, ok := ctx.Deadline()
	if ok {
		aggregateOptions.SetMaxTime(deadline.Sub(time.Now()))
	}
	aggregate, err := s.DbCollection.Aggregate(
		ctx,
		mongo.Pipeline{bson.D{{Key: "$match", Value: matchStageValue}}, sortStage, groupStage, lookupStage},
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
		PrevClosePrice:     parsePrevClosePrice(m["prev_window_trade"]),
	}
}

func parsePrevClosePrice(i interface{}) string {
	a, ok := i.(bson.A)
	if !ok {
		return ""
	}
	if len(a) != 1 {
		return ""
	}
	m, ok := a[0].(bson.M)
	if !ok {
		return ""
	}
	data, ok := m["data"].(bson.M)
	if !ok {
		return ""
	}

	if price, ok := data["price"].(primitive.Decimal128); ok {
		return price.String()
	}

	if price, ok := data["price"].(string); ok {
		return price
	}

	if price, ok := data["price"].(float64); ok {
		return strconv.FormatFloat(price, 'f', -1, 64)
	}

	return ""
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
