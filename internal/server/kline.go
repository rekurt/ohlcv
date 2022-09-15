package server

import (
	"bitbucket.org/novatechnologies/ohlcv/protocol/kline"
	"context"
)

type Kline struct {
	kline.UnimplementedKlineServiceServer
}

func (k Kline) Get(ctx context.Context, request *kline.GetKlineRequest) (*kline.GetKlineResponse, error) {
	//TODO implement me
	panic("implement me")
}

func New() *Kline {
	return &Kline{}
}
