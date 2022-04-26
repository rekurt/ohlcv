package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/domain"
)

type CandleHandler struct {
	CandleService *candle.Service
	Broadcaster   domain.Broadcaster
}

const defaultDuration = 5 * time.Minute

func NewCandleHandler(candleService *candle.Service, broadcaster domain.Broadcaster) *CandleHandler {
	return &CandleHandler{candleService, broadcaster}
}

func (h CandleHandler) GetCandleChart(
	res http.ResponseWriter,
	req *http.Request,
) {
	ctx := req.Context()

	market := req.URL.Query().Get("market")

	resolution := req.URL.Query().Get("resolution")

	candleDuration := domain.StrResolutionToDuration(resolution)
	if candleDuration == 0 {
		candleDuration = defaultDuration
		resolution = domain.Candle5MResolution
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

	chart, _ := h.CandleService.GetChart(ctx, market, resolution, from, to)

	marshal, err := json.Marshal(chart)
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
