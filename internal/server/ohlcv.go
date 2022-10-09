package server

import (
	"context"

	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/internal/consumer"
	"bitbucket.org/novatechnologies/ohlcv/internal/model"
	"bitbucket.org/novatechnologies/ohlcv/internal/service"
	"bitbucket.org/novatechnologies/ohlcv/protocol/ohlcv"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Ohlcv struct {
	candleService *service.Candle
	klineService  *service.Kline
	dealConsumer  *consumer.Deal
	dealService   *service.Deal
	ohlcv.UnimplementedOHLCVServiceServer
}

func NewOhlcv(
	candleService *service.Candle,
	klineService *service.Kline,
	dealService *service.Deal,
	dealConsumer *consumer.Deal,
) *Ohlcv {
	return &Ohlcv{
		candleService: candleService,
		klineService:  klineService,
		dealService:   dealService,
		dealConsumer:  dealConsumer,
	}
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
			Quotes:      klns[i].Quotes.String(),
			OpenTime:    timestamppb.New(klns[i].OpenTime),
			CloseTime:   timestamppb.New(klns[i].CloseTime),
			Trades:      int32(klns[i].Trades),
			TakerQuotes: klns[i].TakerQuotes.String(),
			TakerAssets: klns[i].TakerAssets.String(),
			First:       timestamppb.New(klns[i].First),
			Last:        timestamppb.New(klns[i].Last),
		}
	}
	return rsp, nil
}

func (h Ohlcv) GetLastTrades(ctx context.Context, request *ohlcv.GetLastTradesRequest) (*ohlcv.GetLastTradesResponse, error) {
	trades, err := h.dealService.GetLastTrades(ctx, request.Symbol, request.Limit)
	if err != nil {
		logger.FromContext(ctx).Errorf("error getting last trades: %v", err)
		return nil, err
	}

	rsp := &ohlcv.GetLastTradesResponse{Trades: make([]*ohlcv.Trade, len(trades))}

	for i := range trades {
		rsp.Trades[i] = &ohlcv.Trade{
			Id:           trades[i].Data.DealId,
			Price:        trades[i].Data.Price.String(),
			Qty:          trades[i].Data.Volume.String(),
			QuoteQty:     trades[i].Data.Volume.String(),
			Time:         trades[i].T.Time().UnixNano(),
			IsBuyerMaker: trades[i].Data.IsBuyerMaker,
		}
	}

	return rsp, nil
}

func (h Ohlcv) GetTicker(ctx context.Context, r *ohlcv.GetTickerRequest) (*ohlcv.GetTickerResponse, error) {
	tickers, err := h.dealService.GetTickerPriceChangeStatistics(ctx, r.Symbol)
	if err != nil {
		logger.FromContext(ctx).Errorf("error getting tickers: %v", err)
		return nil, err
	}
	rsp := &ohlcv.GetTickerResponse{Tickers: make([]*ohlcv.Ticker, len(tickers))}
	for i := range tickers {
		rsp.Tickers[i] = &ohlcv.Ticker{
			Symbol:             tickers[i].Symbol,
			PriceChange:        tickers[i].PriceChange,
			PriceChangePercent: tickers[i].PriceChangePercent,
			WeightedAvgPrice:   tickers[i].WeightedAvgPrice,
			PrevClosePrice:     tickers[i].PrevClosePrice,
			LastPrice:          tickers[i].LastPrice,
			LastQty:            tickers[i].LastQty,
			BidPrice:           tickers[i].BidPrice,
			BidQty:             tickers[i].BidQty,
			AskPrice:           tickers[i].AskPrice,
			AskQty:             tickers[i].AskQty,
			OpenPrice:          tickers[i].OpenPrice,
			HighPrice:          tickers[i].HighPrice,
			LowPrice:           tickers[i].LowPrice,
			Volume:             tickers[i].Volume,
			QuoteVolume:        tickers[i].QuoteVolume,
			OpenTime:           tickers[i].OpenTime,
			CloseTime:          tickers[i].CloseTime,
			FirstId:            tickers[i].FirstId,
			LastId:             tickers[i].LastId,
			Count:              int32(tickers[i].Count),
		}
	}
	return rsp, nil

}

func (h Ohlcv) SubscribeDeals(_ *ohlcv.SubscribeDealsRequest, server ohlcv.OHLCVService_SubscribeDealsServer) error {
	log := logger.FromContext(server.Context())
	id, err := uuid.NewUUID()
	if err != nil {
		log.Errorf("can't create uuid %v", err)
		return err
	}
	ch := make(chan *model.Deal, 1024)
	h.dealConsumer.Subscribe(id.String(), ch)
	defer func() {
		h.dealConsumer.UnSubscribe(id.String())
	}()
	for {
		select {
		case <-server.Context().Done():
			return nil
		case d := <-ch:
			err := server.Send(&ohlcv.SubscribeDealsResponse{
				Time:         timestamppb.New(d.T.Time()),
				DealId:       d.Data.DealId,
				Price:        d.Data.Price.String(),
				Volume:       d.Data.Volume.String(),
				Symbol:       d.Data.Market,
				IsBuyerMaker: d.Data.IsBuyerMaker,
			})
			if err != nil {
				logger.FromContext(server.Context()).Errorf("can't send deals %v", err)
			}
		}
	}
}
