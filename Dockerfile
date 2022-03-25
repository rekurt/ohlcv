FROM golang:1.17.2-alpine
RUN mkdir /app
ADD docker /app
WORKDIR /app
COPY docker /app/
COPY . /app/
RUN git config --global url."git@bitbucket.org:".insteadOf "https://bitbucket.org/"
RUN apk add git libc-dev gcc vim && go mod tidy
RUN CGO_ENABLED=1 GO111MODULE=on go build -tags=jsoniter -a -o ohlcv cmd/main.go
CMD ["/app/ohlcv"]
