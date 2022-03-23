package handler

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"encoding/json"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
)

type CandleHandler struct{
	CandleService *candle.Service
}

func NewCandleHandler(candleService *candle.Service) *CandleHandler {
	return &CandleHandler{candleService}
}

var upgrader = websocket.Upgrader{}

func (h CandleHandler) GetUpdatedCandle(w http.ResponseWriter, r *http.Request) {
	ctx := infra.GetContext()
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	quit := make(chan struct{})
	go func() {
		var message domain.Candle
		for {
			select {
			case message = <- h.CandleService.UpdatedCandles:
				encodeM, err := json.Marshal(message)
				logger.FromContext(ctx).Infof("Push candle to websocket")
				err = c.WriteMessage(1, encodeM)
				if err != nil {
					log.Println("write:", err)
					break
				}
			case <- quit:
				return
			}
		}
	}()
}

func (h CandleHandler) GetCandle(res http.ResponseWriter, req *http.Request) {
	ctx := infra.GetContext()

	var market string
	market = req.URL.Query().Get("market")
	req.URL.Query().Get("interval")
	candles, _ := h.CandleService.GetMinuteCandles(ctx, market)
	marshal, err := json.Marshal(candles)
	if err != nil {
		return
	}
	io.WriteString(res, string(marshal))
}
