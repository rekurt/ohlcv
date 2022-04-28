FROM golang:1.17-alpine as builder

# gcc и musl-dev зависимости golangci-lint (make install-tools)
RUN apk add --update make git musl-dev gcc

WORKDIR /app

ARG SVC=main
ARG GOOS=linux
ARG CGO_ENABLED=0

ENV GO111MODULE=on
ENV CGO_ENABLED=$CGO_ENABLED
ENV GOOS=$GOOS
ENV GOBIN_PATH=/app/$SVC

COPY . .

RUN make fmt
RUN make lint
RUN make tests
RUN make build

# Production image
FROM scratch AS prod

ARG SVC=main
ENV APP_NAME="${SVC}"

WORKDIR /app
COPY --from=builder /app/$APP_NAME ./$APP_NAME
COPY --from=builder /app/config/deploy.env ./config/.env

ENTRYPOINT ["/app/$APP_NAME}"]
