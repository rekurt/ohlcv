package server

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/internal/service"
	"bitbucket.org/novatechnologies/ohlcv/protocol/ohlcv"
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Ohlcv struct {
	ohlcv.UnimplementedOHLCVServiceServer
	candleService *service.Candle
}

func NewOhlcv(candleService *service.Candle) *Ohlcv {
	return &Ohlcv{candleService: candleService}
}

// GenerateMinutesCandle returns all minute candles
func (h Ohlcv) GenerateMinutesCandle(ctx context.Context, request *ohlcv.GenerateMinuteCandlesRequest) (*ohlcv.GenerateMinuteCandlesResponse, error) {
	cdls, err := h.candleService.GenerateMinuteCandles(ctx, request.StartTime.AsTime(), request.EndTime.AsTime())
	if err != nil {
		logger.FromContext(ctx).Errorf("can't generate minutes candles %v", err)
		return nil, err
	}
	rsp := &ohlcv.GenerateMinuteCandlesResponse{Candles: make([]*ohlcv.Candle, len(cdls))}
	for i := range cdls {
		rsp.Candles[i] = &ohlcv.Candle{
			Open:     cdls[i].Open.String(),
			High:     cdls[i].High.String(),
			Low:      cdls[i].Low.String(),
			Close:    cdls[i].Close.String(),
			Symbol:   cdls[i].Symbol,
			Volume:   cdls[i].Volume.String(),
			OpenTime: timestamppb.New(cdls[i].OpenTime),
		}
	}
	return rsp, nil
}
