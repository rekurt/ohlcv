PointPay.io Exchange OHLCV data service 

### Для того чтобы скачивались зависимости из приватных bitbucket/gitlab репозиториев
https://novatechnologies.atlassian.net/wiki/spaces/PEO/pages/2345664548/go


### Environment variables
```
LOG_LEVEL=                                                      // trace/debug/info/error/critical. default:"error"

// Доступы к MongoDB
MONGODB_URL
MONGODB_NAME
MONGODB_TIMEOUT
MONGODB_DEAL_COLLECTION_NAME
MONGODB_MINUTE_CANDLE_COLLECTION_NAME
MONGODB_ROOT_PASSWORD

KAFKA_HOST=                                                     // Адреса брокеров кафки через запятую
KAFKA_SSL=                                                 // Используется ли SSL для подключения к кафке
KAFKA_TOPIC_PREFIX=                                             // Префикс для топиков     
KAFKA_CONSUMER_COUNT=

EXCHANGE_MARKETS_SERVER_URL=https://api.exchange.pointpay.io    // url to exchange php backend service
EXCHANGE_MARKETS_TOKEN=                                         // token for exchange php backend service
EXCHANGE_SERVER_SSL=true                                        // Exchange server API use SSL?

// Доступы к Centrifugo
CENTRIFUGE_HOST= centrifugo.xch-master.svc.cluster.local:8000
CENTRIFUGE_TOKEN=
```

### For install:

```bash
go mod tidy -v
go build -tags=jsoniter -a -o ./bin/ohlcv cmd/consumer/main.go
```

### Setup local third party services
For the first time setup:
```bash
make init
```
Usual setup/teardown:
```bash
make {docker-up, docker-stop}
```
#### Setup with deals fixtures
```bash
make db-seed
```
### Clean up local dev environment
```bash
make docker-clean
```