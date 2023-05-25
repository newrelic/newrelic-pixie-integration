include .env

BIN_DIR = ./bin
DOCKER_IMAGE_TAG ?= latest

# required for enabling Go modules inside $GOPATH
export GO111MODULE=on

.PHONY: all
all: lint test build-container build

.PHONY: build
build:
	sh scripts/build.sh

.PHONY: fmt
fmt:
	sh scripts/format.sh

.PHONY: lint
lint:
	-sh scripts/lint.sh

config=env.list
.PHONY: run
run:
	sh scripts/run.sh $(config)

.PHONY: test
test:
	@echo "[test] Running unit tests"
	@go test ./...

.PHONY: setup
setup:
	sh scripts/setup.sh

.PHONY: build-container
build-container:
	sh scripts/build-container.sh $(DOCKER_IMAGE_NAME) $(DOCKER_IMAGE_TAG)
