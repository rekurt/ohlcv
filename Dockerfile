FROM golang:1.17.2-alpine
RUN mkdir /app
ADD docker /app
WORKDIR /app
COPY docker /app/
COPY . /app/
ARG GONOSUMDB="bitbucket.org/novatechnologies"
ARG GOPRIVATE="bitbucket.org/novatechnologies"
ARG GONOPROXY="bitbucket.org/novatechnologies"
ARG CGO_ENABLED=1
ARG GO111MODULE=on
RUN apk add git libc-dev gcc vim make mercurial
RUN git config --global url."git@bitbucket.org:".insteadOf "https://bitbucket.org/"
RUN go env
RUN go mod tidy -v
RUN  go build -tags=jsoniter -a -o ohlcv cmd/main.go
CMD ["/app/ohlcv"]
