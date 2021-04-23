#!/usr/bin/env bash

echo "[lint] running golangci checks..."

GOFLAGS=-mod=vendor go run -mod=vendor \
  github.com/golangci/golangci-lint/cmd/golangci-lint run --verbose