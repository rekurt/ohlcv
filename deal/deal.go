package deal

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"context"
	"go.mongodb.org/mongo-driver/mongo"
)

type Service struct {
	DbCollection *mongo.Collection
}

func NewService(dbCollection *mongo.Collection) *Service {
	return &Service{DbCollection: dbCollection}
}

func (s Service) SaveDeal(ctx context.Context, deal *domain.Deal) (*mongo.InsertOneResult, error) {
	res, err := s.DbCollection.InsertOne(ctx, deal)

	if err != nil {
		logger.FromContext(ctx).Errorf("[DealService]Failed save deal. ", err)

		return nil, err
	}

	return res, err
}
