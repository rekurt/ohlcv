PointPay.io Exchange OHLCV data service 

### For install:

```bash
go mod tidy -v
go build -tags=jsoniter -a -o ./bin/ohlcv cmd/main.go
