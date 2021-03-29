BIN_DIR = ./bin
TOOLS_DIR := $(BIN_DIR)/dev-tools
BINARY_NAME ?= pixie-integration
DOCKER_IMAGE_NAME ?= newrelic/pixie-integration
DOCKER_IMAGE_TAG ?= 1.0.0

GOLANGCILINT_VERSION = 1.33.0

# required for enabling Go modules inside $GOPATH
export GO111MODULE=on

.PHONY: all
all: build

.PHONY: build
build: lint test build-container

$(TOOLS_DIR):
	@mkdir -p $@

$(TOOLS_DIR)/golangci-lint: $(TOOLS_DIR)
	@echo "[tools] Downloading 'golangci-lint'"
	@wget -O - -q https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | BINDIR=$(@D) sh -s v$(GOLANGCILINT_VERSION) > /dev/null 2>&1

.PHONY: lint
lint: $(TOOLS_DIR)/golangci-lint
	@echo "[validate] Validating source code running golangci-lint"
	@$(TOOLS_DIR)/golangci-lint run

.PHONY: build-container
build-container:
	docker build -t $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) .

.PHONY: test
test:
	@echo "[test] Running unit tests"
	@go test ./...

