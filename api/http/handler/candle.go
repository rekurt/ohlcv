package handler

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"encoding/json"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
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


func (h CandleHandler) GetUpdatedCandle(w http.ResponseWriter, r *http.Request) {
	ctx := infra.GetContext()
	c, err := h.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	//defer c.Close()
	quit := make(chan struct{})
	for {
		select {
		case message := <- h.CandleService.UpdatedCandles:
			encodeM, err := json.Marshal(message)
			logger.FromContext(ctx).WithField("CandleTimestamp", message.T[0]).Infof("Push candle to websocket.")
			err = c.WriteMessage(1, encodeM)
			if err != nil {
				log.Println("write:", err)
				break
			}
		case <- quit:
			return
		}
	}
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
