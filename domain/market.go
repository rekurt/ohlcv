package domain

import (
	"strings"

	"bitbucket.org/novatechnologies/ohlcv/client/market"
)

func GetAvailableMarketsMap(markets []market.Market) map[string]string {
	m := map[string]string{}
	for _, v := range markets {
		m[v.ID] = v.Name
	}
	return m
}

func NormalizeMarketName(market string) string {
	market = strings.Replace(market, "%2F", "_", -1)
	market = strings.Replace(market, "/", "_", -1)
	return market
}
