#!/bin/env make

$(shell cp -u config/.env.sample config/.env)
-include config/.env
export

REGISTRY          = bitbucket.org/novatechnologies
CGO_ENABLED       ?= 0
GO                =  go
DOCKER_DIR        =  docker## docker dir (for compose and Dockerfile)
COMPOSE_PFS  ?= $(shell echo ${DCP-db,broker,ws})## docker-compose profiles
SCRIPTS_DIR       ?= $(PWD)/scripts## path to the shell scripts-helpers
BIN_DIR	          ?= $(PWD)/bin## path for the build binaries and 3rd party tools
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

export CFG_FILE          := $(PWD)/config/.env
export PROJECT_ROOT 	 := $(PWD)
export GOOS         	 := $(OSFLAG)
export CGO_ENABLED  	 := $(CGO_ENABLED)
export GOBIN        	 := $(PWD)/$(BIN_DIR)
export PATH         	 := $(GOBIN):$(PATH)
export GO111MODULE  	 := on


.PHONY: init
LINE=vm.overcommit_memory=1
SYSCTL_FILE=/etc/sysctl.conf
init: install-tools gen
	$(info $(M) linking centrifugo config.json from volumes to config)
	rm -f $(PWD)/config/centrifugo.json
	ln -s $(PWD)/docker/.volumes/centrifugo/config.json $(PWD)/config/centrifugo.json

	$(info $(M) cp -u $(SCRIPTS_DIR)/docker-compose.sh $(BIN_DIR)/dc for shortcutting)
	rm -f $(BIN_DIR)/dc
	ln -s $(SCRIPTS_DIR)/docker-compose.sh $(BIN_DIR)/dc
	chmod +x $(SCRIPTS_DIR)/docker-compose.sh
	chmod +x $(BIN_DIR)/dc

	$(info $(M) writing "$(LINE)" to "$(SYSCTL_FILE)" for redis)
	$(shell sudo grep -qF "$(LINE)" "$(SYSCTL_FILE)" || echo "$(LINE)" | sudo tee -a "$(SYSCTL_FILE)")
	@(sudo sysctl vm.overcommit_memory=1)

	$(info $(M) building docker-compose images if needed)
	@(COMPOSE_PROFILES=$(COMPOSE_PFS) $(SCRIPTS_DIR)/docker-compose.sh up -d --build)


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
	@(mkdir -p $(BIN_DIR) && touch $(BIN_DIR)/.gitkeep && git add "$(BIN_DIR)/.gitkeep")
	@(echo "GOBIN=$(GOBIN) BIN_DIR=$(BIN_DIR) ./scripts/install-tools.sh")
	@(GOBIN=$(GOBIN) BIN_DIR=$(BIN_DIR) ./scripts/install-tools.sh)


.PHONY: lint
lint: gen ## Run go generation processes
	$(info $(M) running with config ./config/.golangci.yml)
	$(BIN_DIR)/golangci-lint run --config ./config/.golangci.yml ./...


.PHONY: fmt
fmt: install-tools ## Do code formatting
	$(info $(M) running with goimports and gofumpt tools...)
	$(BIN_DIR)/goimports -l -w .
	$(BIN_DIR)/gofumpt -l -w .


# TODO: docs gen
.PHONY: gen
gen: install-tools api_gen fmt ## Generate code, fixtures, docs etc
	$(info $(M) run code and docs generation)
	@(GOBIN=$(GOBIN) PROJECT_ROOT=$(PROJECT_ROOT) $(GO) generate ./...)


.PHONY: api_gen
api_gen: ## Generates Go code from api/openapi.yaml
	PATH=$(BIN_DIR):${PATH} \
	OPENAPI_GENERATOR_VERSION=6.0.0-beta \
	openapi-generator-cli generate \
		-i api/openapi.yaml -g go-server -o api/generated -p outputAsLibrary=true \
		--minimal-update

	$(MAKE) fmt

	git add api/generated/*


.PHONY: test
test: stop lint ## Run unit and mocked/stubbed integration (fast running) tests
	$(info $(M) running unit and integration tests)
	@$(BIN_DIR)/godotenv -f ./config/testing.env \
		$(GO) test -count=1 -parallel=N -race -cover -short ./...


.PHONY: test-integration
test-integration: stop lint ## Run functional/end-to-end integration (slow running) tests
	$(info $(M) running functional and end-to-end tests)
	@$(BIN_DIR)/godotenv -f ./config/.env.testing \
	$(GO) test -count=1 -parallel=N $(MODULE_NAME)/tests


.PHONY: docker-up
docker-up: ## Starts local dev environment: make docker-up
	$(info $(M) starting local dev environment...)
	@(COMPOSE_PROFILES=$(COMPOSE_PFS) $(SCRIPTS_DIR)/docker-compose.sh \
    		up -d)


.PHONY: docker-stop
docker-stop: ## Stops local dev environment: make docker-stop
	$(info $(M) stopping local dev environment...)
	@(echo $(SCRIPTS_DIR)/docker-compose.sh stop)
	@(COMPOSE_PROFILES=$(COMPOSE_PFS) $(SCRIPTS_DIR)/docker-compose.sh stop)


.PHONY: docker-clean ## Cleaning docker images, footprint: make docker-clean.
docker-clean: docker-stop
	$(info $(M) cleaning local docker environment...)

	@(COMPOSE_PROFILES=$(COMPOSE_PFS) $(SCRIPTS_DIR)/docker-compose.sh \
		down --remove-orphans --rmi local)

	@(sudo rm -rf \
		./docker/.volumes/mongo/db \
		./docker/.volumes/redis/data \
	)

.PHONY: db-clean
db-clean:
	@(COMPOSE_PROFILES=$(COMPOSE_PFS) $(SCRIPTS_DIR)/docker-compose.sh \
		stop mongo)
	@(COMPOSE_PROFILES=$(COMPOSE_PFS) $(SCRIPTS_DIR)/docker-compose.sh \
    		down --remove-orphans --rmi local)
	@(sudo rm -rf ./docker/.volumes/mongo/db)


.PHONY: clean
BIN_BASE=$(shell basename $(BIN_DIR))
clean: docker-clean ## Clean up the DB footprint, artefacts etc.
	$(info $(M) cleaning up generated by init files. You should do to reinit project 'make init')

	@(rm -rf ./$(BIN_BASE)/* ./config/.env ./api/generated/go/* ./config/centrifugo.json)
	@(touch $(BIN_BASE)/.gitkeep && git add $(BIN_BASE)/.gitkeep)


.PHONY: db-seed
db-seed: db-clean ## Seeding database with fixtures
	$(info $(M) seeding local docker environment\: mongo...)
	@(COMPOSE_PROFILES=$(COMPOSE_PFS) $(SCRIPTS_DIR)/docker-compose.sh up -d mongo)

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
