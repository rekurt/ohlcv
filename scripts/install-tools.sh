#!/usr/bin/env bash
set -e

# Fix for the 3rd party tools and binaries dir path.
if [ -z "${BIN_DIR}" ]; then BIN_DIR=$(pwd)/bin; fi

if [[ ! -f "$BIN_DIR"/golangci-lint ]]; then
    GOBIN="$BIN_DIR" go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
fi

if [[ ! -f "$BIN_DIR"/mockgen ]]; then
  GOBIN="$BIN_DIR" go install github.com/golang/mock/mockgen@latest
fi

if [[ ! -f "$BIN_DIR"/goimports ]]; then
  GOBIN="$BIN_DIR" go install golang.org/x/tools/cmd/goimports@latest
fi

if [[ ! -f "$BIN_DIR"/godotenv ]]; then
  GOBIN="$BIN_DIR" go install github.com/joho/godotenv/cmd/godotenv@latest
fi

if [[ ! -f "$BIN_DIR"/gofumpt ]]; then
  GOBIN="$BIN_DIR" go install mvdan.cc/gofumpt@latest
fi

if [[ ! -f "$BIN_DIR"/openapi-generator-cli ]]; then
  gen_cli_url=https://raw.githubusercontent.com/OpenAPITools/openapi-generator/master/bin/utils/openapi-generator-cli.sh
  mvn_url=https://dlcdn.apache.org/maven/maven-3/3.8.5/binaries/apache-maven-3.8.5-bin.zip

  # install Maven
  curl -k -L -s $mvn_url > /tmp/mvn.zip
  unzip /tmp/mvn.zip -d "$BIN_DIR"
  rm /tmp/mvn.zip
  ln -s "$BIN_DIR"/apache-maven-3.8.5/bin/mvn "$BIN_DIR"/mvn && chmod +x "$BIN_DIR"/mvn

  # install OpenAPI Generator Cli
  curl $gen_cli_url > "$BIN_DIR"/openapi-generator-cli
  chmod +x "$BIN_DIR"/openapi-generator-cli
fi

if [[ ! -f "$BIN_DIR"/mongoimport ]]; then
  project_bin=$(pwd)/"$BIN_DIR"
  base_dir=/tmp/mongo-tools
  # shellcheck disable=SC2216
  test "$base_dir" | (cd "$base_dir" && git pull) || \
    git clone https://github.com/mongodb/mongo-tools.git "$base_dir" && cd "$base_dir"

  go mod download && go mod tidy -v
  go build -o "$BIN_DIR"/mongoimport mongoimport/main/mongoimport.go
  go build -o "$BIN_DIR"/mongoexport mongoexport/main/mongoexport.go
  go build -o "$BIN_DIR"/mongostat mongostat/main/mongostat.go
  go build -o "$BIN_DIR"/mongotop mongotop/main/mongotop.go
  go build -o "$BIN_DIR"/mongodump mongodump/main/mongodump.go
  go build -o "$BIN_DIR"/mongorestore mongorestore/main/mongorestore.go

  rm -rf /tmp/mongo-tools
fi

chmod +x "$BIN_DIR"/*