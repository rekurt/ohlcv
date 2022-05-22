package http

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"bitbucket.org/novatechnologies/ohlcv/client/market"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	log "github.com/sirupsen/logrus"

	openapi "bitbucket.org/novatechnologies/ohlcv/api/generated/go"
	"bitbucket.org/novatechnologies/ohlcv/api/http/handler"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/domain"
)

type Server struct {
	srv http.Server
}

func NewServer(candleService *candle.Service, dealService domain.Service, conf infra.Config) *Server {
	mux := http.NewServeMux()

	marketClient, err := market.New(
		market.Config{ServerURL: conf.ExchangeMarketsServerURL, ServerTLS: conf.ExchangeMarketsServerSSL},
		market.NewErrorProcessor(map[string]string{}),
		map[interface{}]market.Option{},
		conf.ExchangeMarketsToken,
	)
	if err != nil {
		log.Fatal("can't market.New:" + err.Error())
	}

	candleHandler := handler.NewCandleHandler(candleService)
	MarketApiService := openapi.NewMarketApiService(dealService, marketClient)
	MarketApiController := openapi.NewMarketApiController(MarketApiService)

	router := openapi.NewRouter(MarketApiController)
	mux.Handle("/", router)
	mux.HandleFunc("/api/candles", candleHandler.GetCandleChart)

	srv := http.Server{
		Addr:    fmt.Sprintf(":%d", conf.HttpConfig.Port),
		Handler: mux,
	}

	serv := &Server{
		srv: srv,
	}

	return serv
}

func (s *Server) Start(ctx context.Context) {
	s.srv.BaseContext = func(listener net.Listener) context.Context {
		return ctx
	}
	go func() {
		log.Info("[*] Http server is started")
		for err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed; {
			log.Info(err)
		}
	}()
}

func (s *Server) Stop(ctx context.Context) {
	if err := s.srv.Shutdown(ctx); err != nil {
		log.Info("shutdown")
	}
}
