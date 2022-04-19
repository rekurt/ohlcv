#!/bin/env make

$(shell cp -u config/example.env config/.env)
-include config/.env
export

REGISTRY          ?= bitbucket.org/novatechnologies
CGO_ENABLED       ?= 0
GO                =  go
DOCKER_DIR        =  docker
BIN_DIR	          ?= ${BIN_DIR:-$(PWD)/bin}## path for the build binaries and 3rd party tools
CFG_FILE          ?= $(PWD)/config/.env## path for configuration files
ALL_SERVICES      ?= $(shell basename -s .docker-compose.yml docker/*.docker-compose.yml)
ALL_COMPOSE_FILES ?= $(addprefix -f ,$(shell ls $(DOCKER_DIR)/*.docker-compose.yml))
ROOT_DIR_NAME     ?= $(shell basename $(PWD))
SVC_NAME          ?= $(shell echo "$(SERVICE_NAME)" | awk '{print tolower($$0)}')
IMAGE_NAME        =  ${REGISTRY}/${SVC_NAME}
MODULE_NAME       =  ${REGISTRY}/${SVC_NAME}
BUILD_DATE        =  $(shell date +"%F %T %Z")
GIT_COMMIT        =  $(shell git rev-parse HEAD)
TAG               ?= $(shell git describe --tags --abbrev=0 2>/dev/null)
M                 =  $(shell printf "\033[34;1m>>\033[0m")

# Detect OS for cross-compilation in the future.
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
	OSFLAG := linux
else
	OSFLAG := osx
endif

# If service name if not specified, use root project dir.
ifeq ($(SVC_NAME),)
	SVC_NAME := $(ROOT_DIR_NAME)
endif

export PROJECT_ROOT := $(PWD)
export GOOS         := $(OSFLAG)
export CGO_ENABLED  := $(CGO_ENABLED)
export GOBIN        := $(BIN_DIR)
export PATH         := $(GOBIN):$(PATH)
export GO111MODULE  := on

# TODO: docs gen
.PHONY: init
init:
	@mkdir -p $(BIN_DIR)
	@($(MAKE) install-tools)

	@ln -sf docker/.volumes/centrifugo/config.json config/centrifugo.json
	@sudo chown -R 1001 docker/.volumes/redis

	@docker-compose --project-directory . --env-file $(CFG_FILE) $(COMPOSE_FILES) build

.PHONY: all
all: init gen test docker-build ## Launch almost all processes to make project full build


.PHONY: build
build: gen ## Build any of services, e.g.: make build SVC_NAME=my_service
	$(info $(M) building $(SVC_NAME) into $(BIN_DIR)/$(SVC_NAME)...)
	$(GO) mod tidy
	$(GO) build -o "$(BIN_DIR)/$(SVC_NAME)" \
		-ldflags '-X "build.Version=${TAG}" -X "build.Date=${BUILD_DATE}" -X "build.GIT_SHA=${GIT_COMMIT}"' \
		./cmd/*.go


build-docker: gen ## Build Docker image, e.g.: make build-docker SVC_NAME=my_service
	@docker build \
		--build-arg "SVC=$(SVC_NAME)" \
		--build-arg "BUILD_DATE=$(BUILD_DATE)" \
		--build-arg "GIT_COMMIT=$(GIT_COMMIT)" \
		--build-arg "GOOS=$(GOOS)" \
		--build-arg "CGO_ENABLED=$(CGO_ENABLED)" \
		-t $(IMAGE_NAME):$(TAG) \
		-t $(IMAGE_NAME):last .


.PHONY: install-tools
install-tools: ## Install tools required to work with the project
	$(info $(M) checking for tools needed)
	@GOBIN=$(BIN_DIR) ./scripts/install-tools.sh


.PHONY: lint
lint: gen ## Run go generation processes
	$(info $(M) running with config ./config/.golangci.yml)
	$(BIN_DIR)/golangci-lint run --config ./config/.golangci.yml ./...


.PHONY: fmt
fmt: install-tools ## Do code formatting
	$(info $(M) running with goimports and gofumpt tools...)
	$(BIN_DIR)/goimports -l -w .
	$(BIN_DIR)/gofumpt -l -w .


.PHONY: gen
gen: install-tools api_gen ## Generate code, fixtures, docs etc
	$(info $(M) run code and docs generation)
	@(GOBIN=$(BIN_DIR) PROJECT_ROOT=$(PROJECT_ROOT) $(GO) generate ./...)


.PHONY: api_gen
api_gen: ## Generates Go code from local/api/openapi.yaml
	docker run --rm -v "${PWD}:/local" openapitools/openapi-generator-cli generate \
		-i local/api/openapi.yaml -g go-server -o local/api/generated --minimal-update


.PHONY: test
test: stop lint ## Run unit and mocked/stubbed integration (fast running) tests
	$(info $(M) running unit and integration tests)
	@$(BIN_DIR)/godotenv -f ./config/testing.env \
		$(GO) test -count=1 -parallel=N -race -cover -short ./...


.PHONY: test-integration
test-integration: stop lint ## Run functional/end-to-end integration (slow running) tests
	$(info $(M) running functional and end-to-end tests)
	@$(BIN_DIR)/godotenv -f ./config/testing.env \
	$(GO) test -count=1 -parallel=N $(MODULE_NAME)/tests


.PHONY: docker-up
docker-up: ## Starts local dev environment: make docker-up [profiles=db,broker,debug,gui]
	$(info $(M) starting local dev environment...)
	$(info $(M) building $(ALL_SERVICES) $(ALL_COMPOSE_FILES))
	@(COMPOSE_PROFILES=$(profiles) docker-compose \
		--project-directory . \
		--env-file $(CFG_FILE) \
		$(ALL_COMPOSE_FILES) \
		up -d)


.PHONY: docker-stop
docker-stop: ## Stops local dev environment: make docker-stop [profiles=db,broker,debug,gui]
	$(info $(M) stopping local dev environment...)
	@(COMPOSE_PROFILES=$(profiles) docker-compose \
		--project-directory . \
		--env-file $(CFG_FILE) \
		$(ALL_COMPOSE_FILES) \
		stop)


.PHONY: docker-clean ## Cleaning docker images, footprint: make docker-clean [svc=mongo,kafka].
docker-clean: docker-stop
	$(info $(M) cleaning local docker environment...)
	@(if [ -z "$(svc)" ]; then \
	    svc='*'; \
	else \
	    svc=$(shell echo "{$(svc)}"); \
	fi)

	@docker-compose \
		--project-directory . \
		--env-file $(CFG_FILE) \
		$(addprefix "-f",$(shell ls $(DOCKER_DIR)/$(svc).docker-compose.yml)) \
		down --remove-orphans --rmi local

	@(sudo rm -rf \
		./docker/.volumes/mongo/db/* \
		./docker/.volumes/redis/data/* \
	)


.PHONY: clean
clean: docker-clean ## Clean up the project artefacts (generated code, binaries, configs and other).
	$(info $(M) cleaning $(BIN_DIR) directory config file and others. You should 'make init')
	@rm -rf $(BIN_DIR)/* config/.env api/generated/go/*


.PHONY: db-seed
db-seed: ## Seeding database with fixtures
	@($(MAKE) docker-clean svc=mongo)
	@($(MAKE) docker-up profiles=db)

	$(info $(M) seeding local docker environment\: mongo...)

	$(BIN_DIR)/mongoimport \
    	--host=${MONGODB_HOST} \
    	--authenticationDatabase=admin \
    	--authenticationMechanism="SCRAM-SHA-256" \
    	--db="${MONGODB_NAME}" \
    	--collection="${MONGODB_DEAL_COLLECTION_NAME}" \
    	--username="${MONGODB_USER}" \
    	--password="${MONGODB_PASSWORD}" \
    	--collection="${MONGODB_DEAL_COLLECTION_NAME}" \
    	--file=./tests/fixtures/deals.min.json \
    	--type=json \
    	--mode=upsert \
    	--numInsertionWorkers=8 \
    	--jsonArray --tlsInsecure --verbose


.PHONY: help
help: ## Show this help
	@grep -E '^[a-z][^:]+:.*?## .*$$' $(MAKEFILE_LIST) | sed "s/Makefile://g" | sed "s/:.*## /::/g" | \
		awk 'BEGIN {FS = "::"}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' | \
		awk 'BEGIN {FS = ":"}; {printf "%s \033[36m%s\033[0m\n", $$1, $$2}'

%:
	true
