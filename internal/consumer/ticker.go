package consumer

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"context"
	"sync"
)

type Ticker struct {
	cache       map[string]*domain.TickerPriceChangeStatistics
	subscribers map[string]chan *matcher.Deal
	mm          map[string]string
	mu          sync.RWMutex
}

func NewTicker(mm map[string]string) *Ticker {
	return &Ticker{
		cache:       make(map[string]*domain.TickerPriceChangeStatistics),
		subscribers: make(map[string]chan *matcher.Deal),
		mm:          mm,
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

func (c *Ticker) GetChanel(key string) (chan *matcher.Deal, bool) {
	c.mu.RLock()

	result, ok := c.subscribers[key]

	c.mu.RUnlock()

	return result, ok
}

func (c *Ticker) Subscribe(key string, channel chan *matcher.Deal) {
	c.mu.Lock()

	c.subscribers[key] = channel

	c.mu.Unlock()
}

func (c *Ticker) UnSubscribe(key string) {
	c.mu.Lock()

	delete(c.subscribers, key)

	c.mu.Unlock()
}

func (c *Ticker) UpdateWithNewDeal(key string, value *matcher.Deal) {
	c.mu.Lock()

	c.subscribers[key] <- value

	c.mu.Unlock()
}

func (c *Ticker) ConsumeNewDeals(ctx context.Context) {
	for _, m := range c.mm {
		ch := make(chan *matcher.Deal, 1024)
		c.Subscribe(m, ch)
		for {
			select {
			case <-ctx.Done():
				return
			case deal := <-ch:
				ticker, ok := c.Get(deal.GetMarket())
				if !ok {
					logger.FromContext(ctx).
						WithField("method", "deal.UpdateTickerFromDeal in consuming").
						WithField("dealMessage", deal).
						Errorf("ticker not found")
					continue
				}
				ticker.LastPrice = deal.GetPrice()
				ticker.LastQty = deal.GetAmount()
				ticker.CloseTime = deal.GetCreatedAt()
				ticker.LastId = deal.GetId()
				c.mu.Lock()
				c.cache[deal.GetMarket()] = ticker
				c.mu.Unlock()
			}
		}
	}
}
