# build binary
FROM golang:1.17.11-alpine AS build

ARG GOOS
ENV CGO_ENABLED=1 \
	GOOS=$GOOS \
	GOARCH=amd64 \
	CGO_CPPFLAGS="-I/usr/include" \
	UID=0 GID=0 \
	CGO_CFLAGS="-I/usr/include" \
	CGO_LDFLAGS="-L/usr/lib -lpthread -lrt -lstdc++ -lm -lc -lgcc -lz " \
	PKG_CONFIG_PATH="/usr/lib/pkgconfig" \
	GO111MODULE=on

RUN apk add --no-cache git make
RUN go get -u golang.org/x/lint/golint


WORKDIR /go/src/bitbucket.org/novatechnologies/ohlcv/
COPY . .

RUN go mod tidy -v
RUN go build -tags=jsoniter -a -o /out/service cmd/consumer/main.go

# copy to alpine image
FROM alpine
WORKDIR /app
COPY --from=build /out/service /app/service
CMD ["/app/service"]
