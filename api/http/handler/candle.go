package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/novatechnologies/common/infra/logger"
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

	if resolution.IsNotExist() {
		http.Error(res, "invalid interval value", http.StatusBadRequest)

		return
	}

	fromStr := req.URL.Query().Get("from")
	toStr := req.URL.Query().Get("to")

	if fromStr == "" || toStr == "" {
		illegalUnixTimestamp(res)

		return
	}

	fromUnix, err := strconv.Atoi(fromStr)
	if err != nil {
		illegalUnixTimestamp(res)

		return
	}

	toUnix, err := strconv.Atoi(toStr)
	if err != nil {
		illegalUnixTimestamp(res)

		return
	}

	from, to := truncateInterval(
		time.Unix(int64(fromUnix), 0),
		time.Unix(int64(toUnix), 0),
		resolution,
	)

	chart := h.CandleService.GetChart(ctx, market, resolution, from, to)

	bytes, err := json.Marshal(chart)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)

		return
	}

	if _, err := res.Write(bytes); err != nil {
		logger.FromContext(ctx).
			Errorf("[CandleHandler_GetCandleChart] error writing response: %s", err)
	}
}

func truncateInterval(from, to time.Time, resolution domain.Resolution) (time.Time, time.Time) {
	if resolution == domain.Candle1MHResolution || resolution == domain.Candle1MH2Resolution {
		from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, from.Location())
		to = time.Date(to.Year(), to.Month()+1, 1, 0, 0, 0, 0, to.Location()).Add(-time.Nanosecond)

		return from, to
	}

	candleDuration := resolution.ToDuration(0)

	from = from.Add(-candleDuration).Truncate(candleDuration)
	to = to.Add(candleDuration).Truncate(candleDuration)

	return from, to
}

func illegalUnixTimestamp(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	msg := "illegal timestamp parameter: must be Unix seconds"
	_, _ = w.Write([]byte(msg))
}
