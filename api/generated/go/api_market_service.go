/*
 * PointPay.io Public Spot API (draft)
 *
 * OpenAPI Specifications for the PointPay.io Public Spot API
 *
 * API version: 0.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

import (
	"context"
	"strings"
	"time"

	"bitbucket.org/novatechnologies/ohlcv/client/market"
	"bitbucket.org/novatechnologies/ohlcv/domain"
)

// MarketApiService is a service that implements the logic for the MarketApiServicer
// This service should implement the business logic for every endpoint for the MarketApi API.
// Include any external packages or services that will be required by this service.
type MarketApiService struct {
	dealService  domain.Service
	marketClient market.Client
}

// NewMarketApiService creates a default api service
func NewMarketApiService(dealService domain.Service, marketClient market.Client) MarketApiServicer {
	return &MarketApiService{dealService: dealService, marketClient: marketClient}
}

// ApiV1TradesGet - Recent Trades List
func (s *MarketApiService) ApiV1TradesGet(
	ctx context.Context,
	symbol string,
	limit int32,
) (ImplResponse, error) {
	if strings.TrimSpace(symbol) == "" || limit <= 0 || limit >= 1000 {
		return Response(400, RespError{}), nil
	}

	trades, err := s.dealService.GetLastTrades(ctx, symbol, limit)
	if err != nil {
		return Response(500, RespError{Msg: err.Error()}), nil
	}
	return Response(200, convertDeals(trades)), nil
}

func (s *MarketApiService) ApiV3Ticker24hrGet(ctx context.Context, market string) (ImplResponse, error) {
	statistics, err := s.dealService.GetTickerPriceChangeStatistics(ctx, time.Hour*24, market)
	if err != nil {
		return Response(500, RespError{}), nil
	}
	return Response(200, convertStatistics(statistics)), nil
}

func convertStatistics(statistics []domain.TickerPriceChangeStatistics) []Ticker {
	tickers := make([]Ticker, len(statistics))
	for i, s := range statistics {
		tickers[i] = Ticker{
			Symbol:             s.Symbol,
			PriceChange:        s.PriceChange,
			PriceChangePercent: s.PriceChangePercent,
			PrevClosePrice:     s.PrevClosePrice,
			LastPrice:          s.LastPrice,
			BidPrice:           s.BidPrice,
			BidQty:             s.BidQty,
			AskPrice:           s.AskPrice,
			AskQty:             s.AskQty,
			OpenPrice:          s.OpenPrice,
			HighPrice:          s.HighPrice,
			LowPrice:           s.LowPrice,
			Volume:             s.Volume,
			QuoteVolume:        s.QuoteVolume,
			OpenTime:           s.OpenTime,
			CloseTime:          s.CloseTime,
			FirstId:            s.FirstId,
			LastId:             s.LastId,
			Count:              int64(s.Count),
		}
	}
	return tickers
}

func convertDeals(tr []domain.Deal) []Trade {
	trades := make([]Trade, len(tr))
	for i := range tr {

		trades[i] = Trade{
			Id:           tr[i].Data.DealId,
			Price:        tr[i].Data.Price.String(),
			Qty:          tr[i].Data.Volume.String(),
			QuoteQty:     tr[i].Data.Volume.String(),
			Time:         tr[i].T.Time().UnixMilli(),
			IsBuyerMaker: tr[i].Data.IsBuyerMaker,
		}
	}
	return trades
}

func (s *MarketApiService) V1TradingStats24hAllGet(ctx context.Context, market string) (ImplResponse, error) {
	statistics, err := s.dealService.GetTickerPriceChangeStatistics(ctx, time.Hour*24, market)
	if err != nil {
		return Response(500, RespError{Msg: "dealService.GetTickerPriceChangeStatistics:" + err.Error()}), nil
	}
	markets, err := s.marketClient.List(ctx)
	if err != nil {
		return Response(500, RespError{Msg: "marketClient.List:" + err.Error()}), nil
	}

	return Response(
		200,
		InlineResponse200{
			Timestamp: time.Now().UnixMilli(),
			Code:      200,
			Success:   "true",
			Data:      convertStatisticsAll(statistics, buildMarketsMap(markets)),
		},
	), nil

}

func buildMarketsMap(markets []market.Market) map[string]market.Market {
	m := map[string]market.Market{}
	for _, mi := range markets {
		m[mi.Name] = mi
	}
	return m
}

func convertStatisticsAll(statistics []domain.TickerPriceChangeStatistics, marketsMap map[string]market.Market) []TickerAll {
	tickers := make([]TickerAll, len(statistics))
	for i, s := range statistics {
		marketInfo := marketsMap[s.Symbol]
		tickers[i] = TickerAll{
			Id:                  marketInfo.ID,
			Market:              s.Symbol,
			LastPrice:           s.LastPrice,
			MakerFee:            marketInfo.MakerFee,
			TakerFee:            marketInfo.TakerFee,
			Precision:           int32(marketInfo.Precision),
			BasePrecision:       int32(marketInfo.BasePrecision),
			QuotedPrecision:     int32(marketInfo.QuotedPrecision),
			OrderMinAmount:      marketInfo.OrderMinAmount,
			OrderMinPrice:       marketInfo.OrderMinPrice,
			OrderMinSize:        marketInfo.OrderMinSize,
			Var24hChange:        s.PriceChange,
			Var24hChangePercent: s.PriceChangePercent,
			Var24hHigh:          s.HighPrice,
			Var24hLow:           s.LowPrice,
			Var24hVolume:        s.Volume,
			BaseCurrency:        marketInfo.BaseCurrency.Symbol,
			QuotedCurrency:      marketInfo.QuotedCurrency.Symbol,
		}
	}
	return tickers
}
