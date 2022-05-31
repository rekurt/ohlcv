package candle

import (
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type CurrentCandle struct {
	Symbol    string
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	OpenTime  time.Time
	CloseTime time.Time
}

func (c CurrentCandle) containsTs(nano int64) bool {
	return c.OpenTime.UnixNano() <= nano && c.CloseTime.UnixNano() > nano
}

type CurrentCandles interface {
	AddDeal(deal matcher.Deal) error
	GetCandle(market, resolution string) CurrentCandle
	GetUpdates() <-chan CurrentCandle
}

var timeNow = func() time.Time {
	return time.Now()
}

type currentCandles struct {
	updatesStream chan CurrentCandle
	candlesLock   sync.Mutex
	candles       map[string]map[string]*CurrentCandle //market-resolution-Candle
	aggregator    Aggregator
}

func NewCurrentCandles(ctx context.Context) CurrentCandles {
	cc := &currentCandles{
		updatesStream: make(chan CurrentCandle),
		candles:       map[string]map[string]*CurrentCandle{},
	}
	go func() {
		ticker := time.NewTicker(time.Minute)
		for {
			select {
			case <-ctx.Done():
				close(cc.updatesStream)
				return
			case <-ticker.C:
				cc.refreshAll()
			}
		}
	}()
	return cc
}

func (c *currentCandles) refreshAll() {
	c.candlesLock.Lock()
	defer c.candlesLock.Unlock()
	for market, resolutions := range c.candles {
		for resolution := range resolutions {
			c.refreshCandle(market, resolution)
		}
	}
}

func (c *currentCandles) AddDeal(deal matcher.Deal) error {
	c.candlesLock.Lock()
	defer c.candlesLock.Unlock()
	for _, resolution := range domain.GetAvailableResolutions() {
		c.refreshCandle(deal.Market, resolution)
		currentCandle := c.candles[deal.Market][resolution]
		if !currentCandle.containsTs(deal.CreatedAt) {
			continue
		}
		err := c.updateCandle(currentCandle, deal)
		if err != nil {
			return fmt.Errorf("can't AddDeal to currentCandles: '%w'", err)
		}
	}
	return nil
}

func (c *currentCandles) refreshCandle(market, resolution string) {
	if m := c.candles[market]; m == nil {
		c.candles[market] = map[string]*CurrentCandle{}
	}
	now := timeNow()
	currentCandle := c.candles[market][resolution]
	if currentCandle == nil || !currentCandle.containsTs(now.UnixNano()) {
		if currentCandle != nil {
			c.updatesStream <- *currentCandle
		}
		openTime := time.Unix(c.aggregator.GetCurrentResolutionStartTimestamp(resolution, now), 0).UTC()
		currentCandle = &CurrentCandle{
			Symbol:    market,
			OpenTime:  openTime,
			CloseTime: openTime.Add(domain.StrResolutionToDuration(resolution)).UTC(),
		}
		c.candles[market][resolution] = currentCandle
		c.updatesStream <- *currentCandle
	}
}

func (c *currentCandles) updateCandle(candle *CurrentCandle, deal matcher.Deal) error {
	dealAmount, err := strconv.ParseFloat(deal.Amount, 64)
	if err != nil {
		return err
	}
	candle.Volume += dealAmount
	dealPrice, err := strconv.ParseFloat(deal.Price, 64)
	if err != nil {
		return err
	}
	if candle.Open == 0 {
		candle.Open = dealPrice
	}
	if dealPrice > candle.High {
		candle.High = dealPrice
	}
	if dealPrice < candle.Low || candle.Low == 0 {
		candle.Low = dealPrice
	}
	return nil
}

func (c *currentCandles) GetUpdates() <-chan CurrentCandle {
	return c.updatesStream
}

func (c *currentCandles) GetCandle(market, resolution string) CurrentCandle {
	c.candlesLock.Lock()
	defer c.candlesLock.Unlock()
	c.refreshCandle(market, resolution)
	return *c.candles[market][resolution]
}
