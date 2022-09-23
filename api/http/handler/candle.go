package handler

import (
	"bitbucket.org/novatechnologies/ohlcv/internal/model"
	"encoding/json"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/novatechnologies/candles/protocol/candle"
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/domain"
)

type CandleHandler struct {
	client candle.CandleServiceClient
}

func NewCandleHandler(candleService candle.CandleServiceClient) *CandleHandler {
	return &CandleHandler{candleService}
}

func (h CandleHandler) GetCandleChart(
	res http.ResponseWriter,
	req *http.Request,
) {
	res.Header().Set("Access-Control-Allow-Origin", req.Header.Get("Origin"))
	res.Header().Set("Access-Control-Allow-Methods", "POST, GET, PATCH, OPTIONS, PUT, DELETE, HEAD")
	res.Header().Set("Access-Control-Allow-Credentials", "true")

	res.Header().Set("Access-Control-Expose-Headers", "Content-Security-Policy, Location")
	res.Header().Set("Access-Control-Allow-Headers", "Content-Length, baggage, Accept-Encoding, X-CSRF-Token, Host, Authorization, sentry-trace, Access-Control-Allow-Headers, Origin, Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers")

	ctx := req.Context()

	market := domain.NormalizeMarketName(req.URL.Query().Get("market"))
	if len(market) == 0 {
		http.Error(res, "market is required", http.StatusBadRequest)

		return
	}

	interval := req.URL.Query().Get("interval")
	resolution := model.Resolution(strings.ToUpper(interval))

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
	unit, unitSize := model.GetResolution(resolution)
	candles, err := h.client.Get(ctx, &candle.GetRequest{
		From:     timestamppb.New(from),
		To:       timestamppb.New(to),
		Symbol:   market,
		Unit:     unit,
		UnitSize: int32(unitSize),
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	rsp := &domain.ChartResponse{
		Symbol: market,
		O:      make([]float64, len(candles.Candles)),
		H:      make([]float64, len(candles.Candles)),
		L:      make([]float64, len(candles.Candles)),
		C:      make([]float64, len(candles.Candles)),
		V:      make([]float64, len(candles.Candles)),
	}

	for i := range candles.Candles {
		c := candles.Candles[i]
		rsp.O[i], err = strconv.ParseFloat(c.Open, 64)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		rsp.H[i], err = strconv.ParseFloat(c.High, 64)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		rsp.L[i], err = strconv.ParseFloat(c.Low, 64)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		rsp.C[i], err = strconv.ParseFloat(c.Close, 64)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		rsp.V[i], err = strconv.ParseFloat(c.Volume, 64)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		rsp.T[i] = c.OpenTime.AsTime().Unix()
	}

	bytes, err := json.Marshal(rsp)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)

		return
	}

	if _, err := res.Write(bytes); err != nil {
		logger.FromContext(ctx).
			Errorf("[CandleHandler_GetCandleChart] error writing response: %s", err)
	}
}

func truncateInterval(from, to time.Time, resolution model.Resolution) (time.Time, time.Time) {
	if resolution == model.Candle1MHResolution || resolution == model.Candle1MH2Resolution {
		from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, from.Location())
		to = time.Date(to.Year(), to.Month()+1, 1, 0, 0, 0, 0, to.Location()).Add(-time.Nanosecond)

		return from, to
	}

	candleDuration := resolution.ToDuration(0, from.Year())

	from = from.Add(-candleDuration).Truncate(candleDuration)
	to = to.Add(candleDuration).Truncate(candleDuration)

	return from, to
}

func illegalUnixTimestamp(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	msg := "illegal timestamp parameter: must be Unix seconds"
	_, _ = w.Write([]byte(msg))
}
