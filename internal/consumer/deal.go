package consumer

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/internal/model"
	"context"
	"sync"
)

type Deal struct {
	dealChan      chan *model.Deal
	subscribers   map[string]chan *model.Deal
	subscribersMu sync.RWMutex
}

func NewDeal(dealChan chan *model.Deal) *Deal {
	return &Deal{
		dealChan:    dealChan,
		subscribers: make(map[string]chan *model.Deal),
	}
}

func (c *Deal) Subscribe(key string, channel chan *model.Deal) {
	c.subscribersMu.Lock()
	c.subscribers[key] = channel
	c.subscribersMu.Unlock()
}

func (c *Deal) UnSubscribe(key string) {
	c.subscribersMu.Lock()
	delete(c.subscribers, key)
	c.subscribersMu.Unlock()
}

func (c *Deal) Consume(ctx context.Context) {
	for {
		select {
		case d := <-c.dealChan:
			c.subscribersMu.RLock()
			for key := range c.subscribers {
				select {
				case c.subscribers[key] <- d:
				default:
					logger.FromContext(ctx).Errorf("channel deals overloaded")
				}
			}
			c.subscribersMu.RUnlock()
		case <-ctx.Done():
			return
		}
	}
}
