package market

import (
	"context"
	"sync"
	"time"
)

type cache struct {
	sync.Mutex
	cli Client
	//last response
	markets []Market
	err     error
}

func NewCache(cli Client) Client {
	c := &cache{cli: cli}
	go func() {
		c.load()
		for range time.Tick(time.Second * 30) {
			c.load()
		}
	}()
	return c
}

func (c *cache) List(_ context.Context) (markets []Market, err error) {
	return c.markets, c.err
}

func (c *cache) load() {
	c.Lock()
	defer c.Unlock()
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelFunc()
	markets, err := c.cli.List(ctx)
	c.markets = markets
	c.err = err
}
