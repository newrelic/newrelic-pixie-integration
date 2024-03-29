include .env

BIN_DIR = ./bin
DOCKER_IMAGE_TAG ?= latest

# required for enabling Go modules inside $GOPATH
export GO111MODULE=on

.PHONY: all
all: test build-container build

.PHONY: build
build:
	sh scripts/build.sh

.PHONY: compile
compile: build

.PHONY: fmt
fmt:
	sh scripts/format.sh

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

buildLicenseNotice:
	@go list -mod=mod -m -json all | go-licence-detector -noticeOut=NOTICE.txt -rules ./assets/license/rules.json  -noticeTemplate ./assets/license/THIRD_PARTY_NOTICES.md.tmpl -noticeOut THIRD_PARTY_NOTICES.md -overrides ./assets/license/overrides -includeIndirect
