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
	GetCandle(market, resolution string) (CurrentCandle, bool)
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
				//try close candle
			}
		}
	}()

	return cc
}

func (c *currentCandles) AddDeal(deal matcher.Deal) error {
	for _, resolution := range domain.GetAvailableResolutions() {
		currentCandle := c.getCurrentCandleOrCreate(deal.Market, resolution)
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

func (c *currentCandles) getCurrentCandleOrCreate(market, resolution string) *CurrentCandle {
	if m := c.candles[market]; m == nil {
		c.candles[market] = map[string]*CurrentCandle{}
	}
	currentCandle := c.candles[market][resolution]
	if currentCandle == nil {
		openTime := time.Unix(c.aggregator.GetCurrentResolutionStartTimestamp(resolution, timeNow()), 0).UTC()
		currentCandle = &CurrentCandle{
			Symbol:    market,
			OpenTime:  openTime,
			CloseTime: openTime.Add(domain.StrResolutionToDuration(resolution)).UTC(),
		}
		c.candles[market][resolution] = currentCandle
	}
	return c.candles[market][resolution]
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

func (c *currentCandles) GetCandle(market, resolution string) (CurrentCandle, bool) {
	candle, ok := c.candles[market][resolution]
	if ok {
		return *candle, true
	}
	return CurrentCandle{}, false
}
