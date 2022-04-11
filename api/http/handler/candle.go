package handler

import (
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"encoding/json"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
)

type CandleHandler struct{
	CandleService *candle.Service
	Upgrader *websocket.Upgrader

}

func NewCandleHandler(candleService *candle.Service) *CandleHandler {
	return &CandleHandler{candleService, &websocket.Upgrader{}}
}

func (h CandleHandler) GetCandleChart(res http.ResponseWriter, req *http.Request) {
	ctx := infra.GetContext()

	var market string
	var interval string
	market = req.URL.Query().Get("market")
	interval = req.URL.Query().Get("interval")
	candles, _ := h.CandleService.GetMinuteCandles(ctx, market)
	result := h.CandleService.AggregateCandleToChartByInterval(candles, interval, 0)
	marshal, err := json.Marshal(result)
	if err != nil {
		return
	}
	io.WriteString(res, string(marshal))
}
