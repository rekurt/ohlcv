
PointPay.io Exchange OHLCV data service 


 
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