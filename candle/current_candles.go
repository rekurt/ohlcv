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

var getAvailableResolutions = func() []string {
	return domain.GetAvailableResolutions()
}

type currentCandles struct {
	updatesStream chan CurrentCandle
	candlesLock   sync.Mutex
	candles       map[string]map[string]*CurrentCandle //market-resolution-Candle
	aggregator    Aggregator
}

func NewCurrentCandles(ctx context.Context) CurrentCandles {
	cc := &currentCandles{
		updatesStream: make(chan CurrentCandle, 512),
		candles:       map[string]map[string]*CurrentCandle{},
	}
	go func() {
		ticker := time.NewTicker(time.Minute) //TODO every round min
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

func (c *currentCandles) setCandle(market, resolution string, candle CurrentCandle) {
	oldCandle := c.getSafeCandle(market, resolution)
	//nothing changed
	if oldCandle != nil && *oldCandle == candle {
		return
	}
	if oldCandle != nil {
		c.updatesStream <- *oldCandle
	}
	c.setSafeCandle(market, resolution, candle)
	c.updatesStream <- candle
}

func (c *currentCandles) refreshAll() {
	c.candlesLock.Lock()
	defer c.candlesLock.Unlock()
	//TODO can we iterate concurrently?
	for market, resolutions := range c.candles {
		for resolution := range resolutions {
			c.setCandle(market, resolution, c.getFreshCandle(market, resolution))
		}
	}
}

func (c *currentCandles) AddDeal(deal matcher.Deal) error {
	c.candlesLock.Lock()
	defer c.candlesLock.Unlock()
	for _, resolution := range getAvailableResolutions() {
		currentCandle := c.getFreshCandle(deal.Market, resolution)
		if !currentCandle.containsTs(deal.CreatedAt) {
			continue
		}
		currentCandle, err := c.updateCandle(currentCandle, deal)
		if err != nil {
			return fmt.Errorf("can't AddDeal to currentCandles: '%w'", err)
		}
		c.setCandle(deal.Market, resolution, currentCandle)
	}
	return nil
}

func (c *currentCandles) getSafeCandle(market, resolution string) *CurrentCandle {
	if c.candles[market] == nil || c.candles[market][resolution] == nil {
		return nil
	}
	return c.candles[market][resolution]
}
func (c *currentCandles) setSafeCandle(market, resolution string, candle CurrentCandle) {
	if c.candles[market] == nil {
		c.candles[market] = map[string]*CurrentCandle{}
	}
	c.candles[market][resolution] = &candle
}
func (c *currentCandles) getFreshCandle(market, resolution string) CurrentCandle {
	now := timeNow()
	candle := c.getSafeCandle(market, resolution)
	if candle == nil || !candle.containsTs(now.UnixNano()) {
		openTime := time.Unix(c.aggregator.GetCurrentResolutionStartTimestamp(resolution, now), 0).UTC()
		return CurrentCandle{
			Symbol:    market,
			OpenTime:  openTime,
			CloseTime: openTime.Add(domain.StrResolutionToDuration(resolution)).UTC(),
		}
	}
	return *candle
}

func (c *currentCandles) updateCandle(candle CurrentCandle, deal matcher.Deal) (CurrentCandle, error) {
	dealAmount, err := strconv.ParseFloat(deal.Amount, 64)
	if err != nil {
		return candle, err
	}
	candle.Volume += dealAmount
	dealPrice, err := strconv.ParseFloat(deal.Price, 64)
	if err != nil {
		return candle, err
	}
	if candle.Open == 0 {
		candle.Open = dealPrice
	}
	candle.Close = dealPrice
	if dealPrice > candle.High {
		candle.High = dealPrice
	}
	if dealPrice < candle.Low || candle.Low == 0 {
		candle.Low = dealPrice
	}
	return candle, nil
}

func (c *currentCandles) GetUpdates() <-chan CurrentCandle {
	return c.updatesStream
}

func (c *currentCandles) GetCandle(market, resolution string) CurrentCandle {
	c.candlesLock.Lock()
	defer c.candlesLock.Unlock()
	return *c.candles[market][resolution]
}
