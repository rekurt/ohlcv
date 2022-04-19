#!/usr/bin/env bash
set -e

# Fix for the 3rd party tools and binaries dir path.
if [ -z "${BIN_DIR}" ]; then BIN_DIR=$(pwd)/third_party; fi

if [[ ! -f "$BIN_DIR/golangci-lint" ]]; then
    go install -mod=readonly github.com/golangci/golangci-lint/cmd/golangci-lint@latest
fi

if [[ ! -f "$BIN_DIR/mockgen" ]]; then
  go install -mod=readonly github.com/golang/mock/mockgen@latest
fi

if [[ ! -f "$BIN_DIR/goimports" ]]; then
  go install -mod=readonly golang.org/x/tools/cmd/goimports@latest
fi

if [[ ! -f "$BIN_DIR/godotenv" ]]; then
  go install -mod=readonly github.com/joho/godotenv/cmd/godotenv@latest
fi

if [[ ! -f "$BIN_DIR/gofumpt" ]]; then
  go install -mod=readonly mvdan.cc/gofumpt@latest
fi

if [[ ! -f "$BIN_DIR/mongoimport" ]]; then
  project_dir=$(pwd)/"$BIN_DIR"
  base_dir=/tmp/mongo-tools
  # shellcheck disable=SC2216
  test "$base_dir" | (cd "$base_dir" && git pull) || \
    git clone https://github.com/mongodb/mongo-tools.git "$base_dir" && cd "$base_dir"

  go mod download && go mod tidy -v
  go build -o bin/mongoimport mongoimport/main/mongoimport.go
  go build -o bin/mongoexport mongoexport/main/mongoexport.go
  go build -o bin/mongostat mongostat/main/mongostat.go
  go build -o bin/mongotop mongotop/main/mongotop.go
  go build -o bin/mongodump mongodump/main/mongodump.go
  go build -o bin/mongorestore mongorestore/main/mongorestore.go

  cp -u "$base_dir"/bin/* "$project_dir"/
  rm -rf /tmp/mongo-tools
fi
