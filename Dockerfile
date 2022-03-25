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
ARG BITBUCKET_USER=$BITBUCKET_USER
ARG BITBUCKET_PASS=$BITBUCKET_PASS
RUN apk add git libc-dev gcc make mercurial openssh
# RUN mkdir ~/.ssh
# COPY /opt/atlassian/pipelines/agent/ssh/id_rsa ~/.ssh/id_rsa
# ARG GIT_SSH_COMMAND="ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no"
# RUN git config --global url."git@bitbucket.org:".insteadOf "https://bitbucket.org/"
# RUN git config --global url."https://$BITBUCKET_USER:$BITBUCKET_PASS@bitbucket.org/".insteadOf "https://bitbucket.org/"
RUN go env
RUN go mod tidy -v
RUN  go build -tags=jsoniter -a -o ohlcv cmd/main.go
CMD ["/app/ohlcv"]
