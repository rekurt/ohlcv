package repository

import (
	"sync"

	"bitbucket.org/novatechnologies/ohlcv/domain"
)

type Ticker struct {
	cache map[string]*domain.TickerPriceChangeStatistics
	mu    sync.RWMutex
}

func NewTicker() *Ticker {
	return &Ticker{
		cache: make(map[string]*domain.TickerPriceChangeStatistics),
	}
}

func (c *Ticker) Get(key string) (*domain.TickerPriceChangeStatistics, bool) {
	c.mu.RLock()

	result, ok := c.cache[key]

	c.mu.RUnlock()

	return result, ok
}

func (c *Ticker) Set(key string, value *domain.TickerPriceChangeStatistics) {
	c.mu.Lock()

	c.cache[key] = value

	c.mu.Unlock()
}
