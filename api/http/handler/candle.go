package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/domain"
)

type CandleHandler struct {
	CandleService *candle.Service
	Upgrader      *websocket.Upgrader
}

const defaultDuration = 5 * time.Minute

func NewCandleHandler(candleService *candle.Service) *CandleHandler {
	return &CandleHandler{candleService, &websocket.Upgrader{}}
}

func (h CandleHandler) GetCandleChart(
	res http.ResponseWriter,
	req *http.Request,
) {
	ctx := req.Context()

	market := req.URL.Query().Get("market")
	market = strings.Replace(market, "/", "_", -1)
	resolution := req.URL.Query().Get("resolution")

	candleDuration := domain.StrIntervalToDuration(resolution)
	if candleDuration == 0 {
		candleDuration = defaultDuration
		resolution = domain.Candle5MInterval
	}

	fromUnix, err := strconv.Atoi(req.URL.Query().Get("from"))
	if err != nil {
		illegalUnixTimestamp(err, res)
		return
	}
	toUnix, err := strconv.Atoi(req.URL.Query().Get("to"))
	if err != nil {
		illegalUnixTimestamp(err, res)
		return
	}

	from := time.Unix(
		int64(fromUnix),
		0,
	).Add(-candleDuration).Truncate(candleDuration)
	to := time.Unix(
		int64(toUnix),
		0,
	).Add(candleDuration).Truncate(candleDuration)
	if to.Sub(from) < 0 || to.Sub(from) > 24*364*5*time.Hour {
		illegalUnixTimestamp(
			fmt.Errorf(
				"requested interfal is incorrect or to big",
			), res,
		)
	}

	candles, _ := h.CandleService.GetMinuteCandles(ctx, market, from, to)
	result := h.CandleService.AggregateCandleToChartByInterval(
		candles, market, resolution, 10,
	)

	marshal, err := json.Marshal(result)
	if err != nil {
		return
	}

	io.WriteString(res, string(marshal))
}

func illegalUnixTimestamp(err error, w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	msg := fmt.Sprintf(
		"illegal timestamp parameter %v: must be Unix seconds", err,
	)
	_, _ = w.Write([]byte(msg))
}
