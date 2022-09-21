package server

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/internal/service"
	"bitbucket.org/novatechnologies/ohlcv/protocol/kline"
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

type Kline struct {
	kline.UnimplementedKlineServiceServer
	klineService *service.Kline
}

// Get the klines by parameters
func (k Kline) Get(ctx context.Context, request *kline.GetKlineRequest) (*kline.GetKlineResponse, error) {
	var startTime, endTime *time.Time
	if request.StartTime != nil {
		t := request.StartTime.AsTime()
		startTime = &t
	}
	if request.EndTime != nil {
		t := request.EndTime.AsTime()
		endTime = &t
	}
	klines, err := k.klineService.Get(ctx, request.Symbol, request.Interval, startTime, endTime, int(request.Limit))
	if err != nil {
		logger.FromContext(ctx).Errorf("can't get klines via GRPC $v", err)
		return nil, err
	}
	rsp := &kline.GetKlineResponse{
		Klines: make([]*kline.Kline, len(klines)),
	}
	for i := range klines {
		rsp.Klines[i] = &kline.Kline{
			OpenPrice:   klines[i].Open.String(),
			ClosePrice:  klines[i].Close.String(),
			TakerAssets: klines[i].TakerAssets.String(),
			TakerQuote:  klines[i].TakerQuotes.String(),
			HighPrice:   klines[i].High.String(),
			LowPrice:    klines[i].Low.String(),
			Volume:      klines[i].Volume.String(),
			Trades:      int32(klines[i].Trades),
			QuoteVolume: klines[i].Quote.String(),
			OpenTime:    timestamppb.New(klines[i].OpenTime),
			CloseTime:   timestamppb.New(klines[i].CloseTime),
		}
	}
	return rsp, nil
}

func NewKline(service *service.Kline) *Kline {
	return &Kline{klineService: service}
}
