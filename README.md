PointPay.io Exchange OHLCV data service 

### For install:

```bash
go mod tidy -v
go build -tags=jsoniter -a -o ./bin/ohlcv cmd/consumer/main.go
```

### Setup local third party services
```bash
make docker-up
```
#### setup with deals fixtures
```bash
make db-seed
```
### Clean up local dev environment
```bash
make docker-clean
```