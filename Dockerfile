FROM golang:1.17.2-alpine
RUN mkdir /app
ADD docker /app
WORKDIR /app
COPY docker /app/
COPY . /app/
ARG GONOSUMDB="bitbucket.org/novatechnologies"
ARG GOPRIVATE="bitbucket.org/novatechnologies"
ARG GONOPROXY="bitbucket.org/novatechnologies"
RUN apk add git libc-dev gcc vim make mercurial
RUN git config --global url."git@bitbucket.org:".insteadOf "https://api.bitbucket.org/"
RUN go env
RUN go -v mod tidy
RUN CGO_ENABLED=1 GO111MODULE=on go build -tags=jsoniter -a -o ohlcv cmd/main.go
CMD ["/app/ohlcv"]
