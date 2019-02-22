# kuzu - GROW server

# The binary to build (just the basename)
BIN := kuzu

# The projects root import path (under GOPATH)
PKG := github.com/thingful/kuzu

# Docker Hub ID to which docker images should be pushed
REGISTRY ?= thingful

# Version string - to be added to the binary
VERSION := $(shell git describe --tags --always --dirty)

# Build date - to be added to the binary
BUILD_DATE := $(shell date -u "+%FT%H:%M:%S%Z")

# Do not change the following variables

PWD := $(shell pwd)

SRC_DIRS := cmd pkg

BASE_IMAGE ?= alpine
BUILD_IMAGE ?= golang:1.11-alpine

IMAGE := $(REGISTRY)/$(BIN)

BUILD_DIRS := bin \
              .go/src/$(PKG) \
							.go/pkg \
							.go/bin/linux_amd64 \
							.go/std/linux_amd64 \
							.go/cache \
							.coverage

all: build

build: bin/$(BIN) ## Build our binary inside a container

bin/$(BIN): $(BUILD_DIRS) .compose
	@echo "--> Building in the containerized environment"
	@docker-compose -f .docker-compose.yml build
	@docker-compose  -f .docker-compose.yml \
		run \
		--rm \
		-u $$(id -u):$$(id -g) \
		--no-deps \
		app \
		/bin/sh -c " \
			VERSION=$(VERSION) \
			PKG=$(PKG) \
			BUILD_DATE=$(BUILD_DATE) \
			BINARY_NAME=$(BIN) \
			./build/build.sh \
		"


shell: .shell ## Open shell in containerized environment
.shell: $(BUILD_DIRS) .compose
	@echo "--> Launching shell in the containerized environment"
	@docker-compose -f .docker-compose.yml \
		run \
		--rm \
		-u "$$(id -u):$$(id -g)" \
		app \
		/bin/sh

.PHONY: test
test: $(BUILD_DIRS) .compose ## Run tests in the containerized environment
	@echo "--> Running tests in the containerized environment"
	@docker-compose -f .docker-compose.yml \
		run \
		--rm \
		-u $$(id -u):$$(id -g) \
		-e "KUZU_DATABASE_URL=postgres://kuzu:password@postgres/kuzu_test?sslmode=disable" \
		app \
		/bin/sh -c " \
			./build/test.sh $(SRC_DIRS) \
		"

.PHONY: test-shell
test-shell: .test-shell ## Open shell in test containerized environment
.test-shell: $(BUILD_DIRS) .compose
	@echo "--> Launching shell in the containerized environment"
	@docker-compose -f .docker-compose.yml \
		run \
		--rm \
		-u "$$(id -u):$$(id -g)" \
		-e "KUZU_DATABASE_URL=postgres://kuzu:password@postgres/kuzu_test?sslmode=disable" \
		app \
		/bin/sh

DOTFILE_IMAGE = $(subst :,_,$(subst /,_,$(IMAGE))-$(VERSION))

container: .container-$(DOTFILE_IMAGE) container-name ## Create delivery container image
.container-$(DOTFILE_IMAGE): bin/$(BIN) Dockerfile.in
	@sed \
		-e 's|ARG_BIN|$(BIN)|g' \
		-e 's|ARG_FROM|$(BASE_IMAGE)|g' \
		Dockerfile.in > .dockerfile-in
	@docker build -t $(IMAGE):$(VERSION) -f .dockerfile-in .
	@docker images -q $(IMAGE):$(VERSION) > $@

.PHONY: container-name
container-name: ## Show the name of the delivery container
	@echo "  container: $(IMAGE):$(VERSION)"

.PHONY: .compose
.compose: ## Create environment specific compose file
	@sed \
		-e 's|ARG_FROM|$(BUILD_IMAGE)|g' \
		-e 's|ARG_WORKDIR|/go/src/$(PKG)|g' \
		Dockerfile.dev > .dockerfile-dev
	@sed \
		-e 's|ARG_DOCKERFILE|.dockerfile-dev|g' \
		-e 's|ARG_IMAGE|$(IMAGE)-dev:$(VERSION)|g' \
		-e 's|ARG_PWD|$(PWD)|g' \
		-e 's|ARG_PKG|$(PKG)|g' \
		-e 's|ARG_BIN|$(BIN)|g' \
		docker-compose.yml > .docker-compose.yml

$(BUILD_DIRS): ## creates build directories
	@mkdir -p $@

.PHONY: version
version: ## returns the current version
	@echo Version: $(VERSION) - $(BUILD_DATE) $(IMAGE)

.PHONY: push
push: .push-$(DOTFILE_IMAGE) push-name
.push-$(DOTFILE_IMAGE):
	@docker push $(IMAGE):$(VERSION)
	@docker images -q $(IMAGE):$(VERSION) > $@

.PHONY: push-name
push-name:
	@echo "  pushed $(IMAGE):$(VERSION)"

.PHONY: start
start: .compose ## start compose services
	@docker-compose -f .docker-compose.yml \
		up

.PHONY: teardown
teardown: .compose ## teardown compose services
	@docker-compose -f .docker-compose.yml \
		down -v

.PHONY: clean
clean: container-clean bin-clean ## remove all artefacts

.PHONY: container-clean
container-clean: ## clean container artefacts
	rm -rf .container-* .dockerfile-* .docker-compose-* .push-*

.PHONY: bin-clean
bin-clean: ## remove generated build artefacts
	rm -rf .go bin .coverage

.PHONY: psql
psql: .compose
	@docker-compose -f .docker-compose.yml start postgres
	@sleep 1
	@docker exec -it kuzu_postgres_1 psql -U postgres kuzu_development
