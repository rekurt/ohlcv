package deal

import (
	"context"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type service struct {
	DbCollection *mongo.Collection
	Markets      map[string]string
}

func NewService(dbCollection *mongo.Collection, markets map[string]string) domain.Service {
	return &service{DbCollection: dbCollection, Markets: markets}
}

func (s service) SaveDeal(ctx context.Context, dealMessage matcher.Deal) (*domain.Deal, error) {
	if dealMessage.TakerOrderId == "" || dealMessage.MakerOrderId == "" {
		logger.FromContext(ctx).Infof("The deal have empty TakerOrderId or MakerOrderId field. Skip. Dont save to mongo.")
		return nil, nil
	}
	floatVolume, _ := strconv.ParseFloat(dealMessage.Amount, 64)
	floatPrice, _ := strconv.ParseFloat(dealMessage.Price, 64)

	marketName := s.Markets[dealMessage.Market]
	deal := &domain.Deal{
		Price:        floatPrice,
		Volume:       floatVolume,
		DealId:       dealMessage.Id,
		Market:       marketName,
		Time:         time.Unix(0, dealMessage.CreatedAt),
		IsBuyerMaker: dealMessage.IsBuyerMaker,
	}

	_, err := s.DbCollection.InsertOne(ctx, deal)

	if err != nil {
		logger.FromContext(ctx).WithField("error", err.Error()).Errorf("[DealService]Failed save deal.")
		return nil, err
	}

	return deal, nil
}

func (s service) GetLastTrades(ctx context.Context, symbol string, limit int32) ([]domain.Deal, error) {
	if strings.TrimSpace(symbol) == "" || limit <= 0 || limit >= 1000 {
		logger.FromContext(ctx).Infof("Incorrect args: symbol='%s', limit=%d", symbol, limit)
		return nil, nil
	}
	cursor, err := s.DbCollection.Find(ctx, bson.M{"market": symbol}, options.Find().SetLimit(int64(limit)))
	if err != nil {
		logger.FromContext(ctx).WithField("error", err.Error()).Errorf("[DealService]Failed GetLastTrades")
		return nil, err
	}
	var deals []domain.Deal
	err = cursor.All(ctx, &deals)
	if err != nil {
		logger.FromContext(ctx).WithField("error", err.Error()).Errorf("[DealService]Failed GetLastTrades")
		return nil, err
	}
	return deals, nil
}
