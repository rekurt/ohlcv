package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/domain"
)

type CandleHandler struct {
	CandleService *candle.Service
}

const defaultDuration = 1 * time.Minute

const defaultBarsCount = 32

func NewCandleHandler(candleService *candle.Service) *CandleHandler {
	return &CandleHandler{candleService}
}

func getDefaultTimeRange(candleDuration time.Duration) (time.Time, time.Time) {
	to := time.Now().Truncate(candleDuration)
	from := to.Add(-(candleDuration * defaultBarsCount))
	return from, to
}

func (h CandleHandler) GetCandleChart(
	res http.ResponseWriter,
	req *http.Request,
) {
	ctx := req.Context()

	market := normalizeMarketName(req.URL.Query().Get("market"))
	if len(market) == 0 {
		http.Error(res, "market is required", http.StatusBadRequest)
		return
	}

	candleDuration, resolution := getCandlesConfig(req.URL.Query().Get("resolution"))
	from, to := getDefaultTimeRange(candleDuration)

	if req.URL.Query().Get("to") != "" || req.URL.Query().Get("from") != "" {
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

		from = time.Unix(
			int64(fromUnix),
			0,
		).Add(-candleDuration).Truncate(candleDuration)

		to = time.Unix(
			int64(toUnix),
			0,
		).Add(candleDuration).Truncate(candleDuration)

		if to.Sub(from) < 0 {
			illegalUnixTimestamp(
				fmt.Errorf(
					"requested interval is incorrect",
				), res,
			)
		}
	}

	chart, _ := h.CandleService.GetChart(ctx, market, resolution, from, to)
	marshal, err := json.Marshal(chart)
	if err != nil {
		return
	}

	io.WriteString(res, string(marshal))
}

func normalizeMarketName(market string) string {
	market = strings.Replace(market, "%2F", "_", -1)
	market = strings.Replace(market, "/", "_", -1)
	return market
}

func getCandlesConfig(resolution string) (time.Duration, string) {

	candleDuration := domain.StrResolutionToDuration(resolution)

	if candleDuration == 0 {
		candleDuration = defaultDuration
		resolution = domain.Candle5MResolution
	}
	return candleDuration, resolution
}

func illegalUnixTimestamp(err error, w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	msg := fmt.Sprintf(
		"illegal timestamp parameter %v: must be Unix seconds", err,
	)
	_, _ = w.Write([]byte(msg))
}
