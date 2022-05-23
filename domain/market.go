package domain

import "strings"

func GetAvailableMarkets() map[string]string {
	return map[string]string{
		"42e98b73-ec0c-4185-b0db-ffc8610f5741": "BTC_XRP",
		"8177b9c1-dd10-49b6-800a-235e429c97dd": "BTC_BCH",
		"b8e3bfce-0b1e-4eb3-9b62-e0fd9b80ada4": "BTC_ETH",
		"e9a4ed4e-75cb-43c5-9f61-8bd97b63fb23": "BTC_LINK",
		"f59eecfd-db38-4f29-b854-e869e056b7d9": "BTC_TRX",
		"bb751ef0-34de-4495-8f43-9d7489371318": "BTC_UNI",
		"72793962-bab5-41a4-9c86-ad79ff984d2d": "BTC_LTC",

		"18da3c5f-7fa2-41cd-b053-00727fececdc": "ETH_TRX",
		"e269c058-3a86-481b-87a3-85fad2d0c74d": "ETH_LINK",
		"86c3be16-b5ce-4b12-8704-80a2b283a26e": "ETH_LTC",
		"6229966b-0e92-4c00-acd7-e5f827cfed05": "ETH_XRP",
		"f246fc92-253d-4e60-a307-658162043543": "ETH_UNI",

		"352656ec-4ad4-4e8b-8dc4-2ddd3e7643b1": "USDT_BTC",
		"637549dd-48a6-4817-8d7b-2c0428dab380": "USDT_LINK",
		"0bd99aa2-a90e-49b2-b9ff-64b7d49b0fc5": "USDT_UNI",
		"c2ac3e7b-76ac-446f-9492-9456b1808858": "USDT_XLM",
		"ec45253b-edb6-48b8-8c0d-8b32c2bf3af0": "USDT_XRP",
		"af8327a8-2599-44b9-9813-8fb3dd236fb0": "USDT_TRX",
		"7ea7fbf5-cd49-4de7-a432-9806004e018d": "USDT_ETH",
		"17f0f56f-7979-4a21-99df-f74f25ac56d4": "USDT_LTC",
		"15954774-df52-4393-bb87-95b1f1e149e3": "USDT_BCH",

		"5e4d67d9-a00d-45e8-8c98-e0dfe4824d73": "DAI_ETH",
		"decb0844-7f4a-456c-a09e-aa5457b50ac1": "DAI_BTC",

		"11a58dbc-15c5-4d45-9e65-d3dc2ff4ea62": "USDC_ETH",
		"bf5934f0-580b-4e1e-9060-27c48b7b275f": "USDC_BTC",
		"c5e57785-9e7d-44a3-8fb0-c30839950276": "ETH_XLM",
		"03b26b3a-3c57-4ff6-a9f8-ea7cc18dae0f": "BTC_XLM",
		"d456718a-0cb0-4363-a100-9ed30b113f8c": "USDT_BNB",
	}
}

func NormalizeMarketName(market string) string {
	market = strings.Replace(market, "%2F", "_", -1)
	market = strings.Replace(market, "/", "_", -1)
	return market
}
