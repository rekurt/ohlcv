package deal

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"strconv"
	"time"
)

type Service struct {
	DbCollection *mongo.Collection
}

func NewService(dbCollection *mongo.Collection) *Service {
	return &Service{DbCollection: dbCollection}
}

func (s Service) SaveDeal(ctx context.Context, dealMessage matcher.Deal) (*mongo.InsertOneResult, error) {
	if dealMessage.TakerOrderId == "" || dealMessage.MakerOrderId == "" {
		logger.FromContext(ctx).Infof("The deal have empty TakerOrderId or MakerOrderId field. Skip. Dont save to mongo.")
		return nil, nil
	}
	floatVolume, _ := strconv.ParseFloat(dealMessage.Amount, 64)
	floatPrice, _ := strconv.ParseFloat(dealMessage.Price, 64)

	deal := &domain.Deal{
		Price:  floatPrice,
		Volume: floatVolume,
		DealId: dealMessage.Id,
		Market: dealMessage.Market,
		Time:   time.Unix(dealMessage.CreatedAt, 0).Truncate(time.Minute),
	}

	res, err := s.DbCollection.InsertOne(ctx, deal)

	if err != nil {
		logger.FromContext(ctx).Errorf("[DealService]Failed save deal. ", err)

		return nil, err
	}

	return res, err
}
