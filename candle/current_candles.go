package candle

import (
	"bitbucket.org/novatechnologies/ohlcv/internal/model"
	"context"
	"fmt"
	"sync"
	"time"

	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CurrentCandles interface {
	AddDeal(deal *matcher.Deal) error
	AddCandle(market string, resolution model.Resolution, candle domain.Candle) error
}

var timeNow = func() time.Time {
	return time.Now()
}

type currentCandles struct {
	updatesStream chan domain.Candle
	candlesLock   sync.Mutex
	candles       map[string]map[model.Resolution]*domain.Candle //market-resolution-Candle, invariant: Candle is always fresh (now in [openTime;closeTime)
	aggregator    Aggregator
	lgr           logger.Logger
}

func NewCurrentCandles(ctx context.Context, updatesStream chan domain.Candle) CurrentCandles {
	cc := &currentCandles{
		updatesStream: updatesStream,
		candles:       map[string]map[model.Resolution]*domain.Candle{},
		aggregator:    Aggregator{},
		lgr:           logger.FromContext(ctx),
	}
	go func() {
		<-ctx.Done()
		close(cc.updatesStream)
	}()
	cro := cron.New(cron.WithLocation(time.UTC), cron.WithSeconds())
	_, err := cro.AddFunc("0 * * * * *", func() {
		cc.refreshAll()
	})
	if err != nil {
		panic(fmt.Errorf("can't build NewCurrentCandles: '%w'", err))
	}
	cro.Start()
	return cc
}

func (c *currentCandles) refreshAll() {
	c.candlesLock.Lock()
	defer c.candlesLock.Unlock()
	for market, resolutions := range c.candles {
		for resolution := range resolutions {
			oldCandle := c.getSafeCandle(market, resolution)
			newCandle := c.getFreshCandle(market, resolution)
			//inherit ohlc values from previous candle
			newCandle.Open = oldCandle.Close
			newCandle.High = oldCandle.Close
			newCandle.Close = oldCandle.Close
			newCandle.Low = oldCandle.Close
			c.setCandle(market, resolution, newCandle, true)
		}
	}
}

func (c *currentCandles) AddCandle(market string, resolution model.Resolution, candle domain.Candle) error {
	c.candlesLock.Lock()
	defer c.candlesLock.Unlock()
	if candle == (domain.Candle{}) {
		candle = c.buildFreshCandle(market, resolution)
	}
	c.setCandle(market, resolution, candle, false)
	//TODO check is it fresh
	return nil
}

func (c *currentCandles) AddDeal(deal *matcher.Deal) error {
	c.candlesLock.Lock()
	defer c.candlesLock.Unlock()
	resolutions := c.candles[deal.Market]
	if len(resolutions) == 0 {
		c.lgr.WithField("m", deal.Market).Infof("absent currentCandle")
	}
	for resolution := range resolutions {
		currentCandle := c.getFreshCandle(deal.Market, resolution)
		if !currentCandle.ContainsTs(deal.CreatedAt) {
			continue
		}
		currentCandle, err := updateCandle(currentCandle, deal)
		if err != nil {
			return fmt.Errorf("can't AddDeal to currentCandles: '%w'", err)
		}
		c.setCandle(deal.Market, resolution, currentCandle, false)
	}
	return nil
}
func (c *currentCandles) setCandle(market string, resolution model.Resolution, candle domain.Candle, isRefresh bool) {
	oldCandle := c.getSafeCandle(market, resolution)
	//nothing changed
	if oldCandle != nil && *oldCandle == candle {
		return
	}
	if oldCandle != nil && isRefresh { //send old candle only on refresh (because it is closed)
		c.updatesStream <- *oldCandle
	}
	c.setSafeCandle(market, resolution, candle)
	c.updatesStream <- candle
}

func (c *currentCandles) getSafeCandle(market string, resolution model.Resolution) *domain.Candle {
	if c.candles[market] == nil || c.candles[market][resolution] == nil {
		return nil
	}
	return c.candles[market][resolution]
}
func (c *currentCandles) setSafeCandle(market string, resolution model.Resolution, candle domain.Candle) {
	if c.candles[market] == nil {
		c.candles[market] = map[model.Resolution]*domain.Candle{}
	}
	c.candles[market][resolution] = &candle
}
func (c *currentCandles) getFreshCandle(market string, resolution model.Resolution) domain.Candle {
	now := timeNow()
	candle := c.getSafeCandle(market, resolution)
	if candle == nil || !candle.ContainsTs(now.UnixNano()) {
		return c.buildFreshCandle(market, resolution)
	}
	return *candle
}

func (c *currentCandles) buildFreshCandle(market string, resolution model.Resolution) domain.Candle {
	openTime := time.Unix(c.aggregator.GetResolutionStartTimestampByTime(resolution, timeNow()), 0).UTC()
	return domain.Candle{
		Symbol:     market,
		Resolution: resolution,
		OpenTime:   openTime,
		CloseTime:  model.CalculateCloseTime(openTime, resolution),
	}
}

func updateCandle(candle domain.Candle, deal *matcher.Deal) (domain.Candle, error) {
	dealPrice, err := primitive.ParseDecimal128(deal.Price)
	if err != nil {
		return domain.Candle{}, err
	}
	if candle.Volume.IsZero() {
		candle.Open = dealPrice
	}
	dealAmount, err := primitive.ParseDecimal128(deal.Amount)
	if err != nil {
		return domain.Candle{}, err
	}
	volume, err := addPrimitiveDecimal128(dealAmount, candle.Volume)
	if err != nil {
		return domain.Candle{}, err
	}
	candle.Volume = volume
	candle.Close = dealPrice
	highCmp, err := compareDecimal128(dealPrice, candle.High)
	if err != nil {
		return domain.Candle{}, err
	}
	if highCmp > 0 {
		candle.High = dealPrice
	}
	lowCmp, err := compareDecimal128(dealPrice, candle.Low)
	if err != nil {
		return domain.Candle{}, err
	}
	if lowCmp < 0 || candle.Low.IsZero() {
		candle.Low = dealPrice
	}
	return candle, nil
}
