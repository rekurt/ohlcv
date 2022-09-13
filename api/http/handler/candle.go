package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
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

func setupCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, sentry-trace")
}

func (h CandleHandler) GetCandleChart(
	res http.ResponseWriter,
	req *http.Request,
) {
	setupCORS(&res)
	ctx := req.Context()
	market := domain.NormalizeMarketName(req.URL.Query().Get("market"))
	if len(market) == 0 {
		http.Error(res, "market is required", http.StatusBadRequest)
		return
	}

	interval := req.URL.Query().Get("interval")
	resolution := domain.Resolution(strings.ToUpper(interval))

	log.Println(resolution)
	candleDuration, resolution := getCandlesConfig(resolution)
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
	log.Println(from, to)

	chart := h.CandleService.GetChart(ctx, market, resolution, from, to)

	marshal, err := json.Marshal(chart)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)

		return
	}

	io.WriteString(res, string(marshal))
}

func parseInterval(fromString, toString string, resolution domain.Resolution) (from, to time.Time, err error) {
	fromUnix, err := strconv.Atoi(fromString)
	if err != nil {
		return from, to, err
	}

	toUnix, err := strconv.Atoi(toString)
	if err != nil {
		return from, to, err
	}

	from = time.Unix(int64(fromUnix), 0)
	to = time.Unix(int64(toUnix), 0)

	if resolution == domain.Candle1MHResolution || resolution == domain.Candle1MH2Resolution {
		from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, from.Location())
		to = time.Date(to.Year(), to.Month()+1, 1, 0, 0, 0, 0, to.Location()).Add(time.Nanosecond)

		return from, to, err
	}

	from.Add(-candleDuration).Truncate(candleDuration)
	to.Add(-candleDuration).Truncate(candleDuration)

	return
}

func getCandlesConfig(resolution domain.Resolution) (time.Duration, domain.Resolution) {
	candleDuration := resolution.ToDuration(0)

	if candleDuration == 0 {
		candleDuration = defaultDuration
		resolution = domain.Candle1MResolution
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
