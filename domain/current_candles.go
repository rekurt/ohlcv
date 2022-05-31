package domain

import (
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"bitbucket.org/novatechnologies/ohlcv/candle"
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
	Timestamp time.Time
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
	aggregator    candle.Aggregator
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
	if m := c.candles[deal.Market]; m == nil {
		c.candles[deal.Market] = map[string]*CurrentCandle{}
	}
	for _, resolution := range GetAvailableResolutions() {
		currentCandle := c.candles[deal.Market][resolution]
		if currentCandle == nil {
			dealPrice, err := strconv.ParseFloat(deal.Price, 64)
			if err != nil {
				return err
			}
			currentCandle = &CurrentCandle{
				Symbol:    deal.Market,
				Open:      dealPrice,
				Timestamp: time.Unix(c.aggregator.GetCurrentResolutionStartTimestamp(resolution, timeNow()), 0),
			}
			c.candles[deal.Market][resolution] = currentCandle
		}
		err := c.updateCandle(currentCandle, deal, currentCandle.Timestamp.Add(StrResolutionToDuration(resolution)))
		if err != nil {
			return fmt.Errorf("can't AddDeal to currentCandles: '%w'", err)
		}
	}
	return nil
}

func (c *currentCandles) updateCandle(candle *CurrentCandle, deal matcher.Deal, closeTime time.Time) error {
	dealAmount, err := strconv.ParseFloat(deal.Amount, 64)
	if err != nil {
		return err
	}
	candle.Volume += dealAmount
	dealPrice, err := strconv.ParseFloat(deal.Price, 64)
	if err != nil {
		return err
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
