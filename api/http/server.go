package http

import (
	"context"
	"fmt"
	"net"
	"net/http"

	log "github.com/sirupsen/logrus"

	openapi "bitbucket.org/novatechnologies/ohlcv/api/generated/go"
	"bitbucket.org/novatechnologies/ohlcv/api/http/handler"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/domain"
)

type Server struct {
	srv http.Server
}

func NewServer(
	candleService *candle.Service,
	dealService domain.Service,
) *Server {
	mux := http.NewServeMux()

	candleHandler := handler.NewCandleHandler(candleService)
	MarketApiService := openapi.NewMarketApiService(dealService)
	MarketApiController := openapi.NewMarketApiController(MarketApiService)

	router := openapi.NewRouter(MarketApiController)
	mux.Handle("/", router)
	mux.HandleFunc("/api/candles", candleHandler.GetCandleChart)

	srv := http.Server{
		Addr:    fmt.Sprintf(":%d", 8082),
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
