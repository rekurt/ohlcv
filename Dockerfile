FROM golang:1.17.2-alpine as builder
RUN apk add git libc-dev gcc
WORKDIR /app
COPY . /app/
RUN CGO_ENABLED=1 GO111MODULE=on go mod tidy
RUN CGO_ENABLED=1 GO111MODULE=on go build -tags=jsoniter -a -o ohlcv cmd/main.go


FROM alpine:3.15 as release
WORKDIR /app
COPY --from=builder /app/ohlcv .

CMD ["/app/ohlcv"]
