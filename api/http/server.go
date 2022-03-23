package http

import (
	"bitbucket.org/novatechnologies/ohlcv/api/http/handler"
	"bitbucket.org/novatechnologies/ohlcv/candle"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
)

type Server struct {
	srv http.Server
}

func NewServer(candleService *candle.Service) *Server {
	mux := http.NewServeMux()

	candleHandler := handler.NewCandleHandler(candleService)
	mux.HandleFunc("/api/candles", candleHandler.GetCandle)
	mux.HandleFunc("/api/ws/candles", candleHandler.GetUpdatedCandle)

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
			log.Info("listen and serve")
		}
	}()
}

func (s *Server) Stop(ctx context.Context) {
	if err := s.srv.Shutdown(ctx); err != nil {
		log.Info("shutdown")
	}
}
