package server

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/internal/service"
	"bitbucket.org/novatechnologies/ohlcv/protocol/ohlcv"
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Ohlcv struct {
	candleService *service.Candle
	klineService  *service.Kline
	ohlcv.UnimplementedOHLCVServiceServer
}

func NewOhlcv(candleService *service.Candle, klineService *service.Kline) *Ohlcv {
	return &Ohlcv{candleService: candleService, klineService: klineService}
}

// GenerateMinutesCandle returns all minute candles
func (h Ohlcv) GenerateMinutesCandle(ctx context.Context, request *ohlcv.GenerateMinuteCandlesRequest) (*ohlcv.GenerateMinuteCandlesResponse, error) {
	cdls, err := h.candleService.GenerateMinuteCandles(ctx, request.From.AsTime(), request.To.AsTime())
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

func (h Ohlcv) GenerateMinutesKlines(ctx context.Context, request *ohlcv.GenerateMinuteKlinesRequest) (*ohlcv.GenerateMinuteKlinesResponse, error) {
	klns, err := h.klineService.Get(ctx, request.From.AsTime(), request.To.AsTime())
	if err != nil {
		logger.FromContext(ctx).Errorf("can't generate minutes klines %v", err)
		return nil, err
	}
	rsp := &ohlcv.GenerateMinuteKlinesResponse{Klines: make([]*ohlcv.Kline, len(klns))}
	for i := range klns {
		rsp.Klines[i] = &ohlcv.Kline{
			Open:        klns[i].Open.String(),
			High:        klns[i].High.String(),
			Low:         klns[i].Low.String(),
			Close:       klns[i].Close.String(),
			Symbol:      klns[i].Symbol,
			Volume:      klns[i].Volume.String(),
			QuoteVolume: klns[i].Volume.String(),
			OpenTime:    timestamppb.New(klns[i].OpenTime),
			CloseTime:   timestamppb.New(klns[i].CloseTime),
			Trades:      int32(klns[i].Trades),
			TakerQuotes: klns[i].TakerQuotes.String(),
			TakerAssets: klns[i].TakerAssets.String(),
		}
	}
	return rsp, nil
}
