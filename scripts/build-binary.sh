#!/usr/bin/env bash
set -e

go mod tidy
GOOS=$GOOS CGO_ENABLED=$CGO_ENABLED go build -o "$BIN_DIR/$SVC_NAME" \
  -ldflags '-X "build.Version=${TAG}" -X "build.Date=${BUILD_DATE}" -X "build.GIT_SHA=${GIT_COMMIT}"' \
  cmd/*.go
