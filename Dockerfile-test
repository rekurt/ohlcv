FROM alpine:3.15 as release
WORKDIR /app
COPY ohlcv .
COPY ./config/.env ./config/.env 

CMD ["/app/ohlcv"]
