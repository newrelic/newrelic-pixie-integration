#!/usr/bin/env bash

echo "[lint] running golangci checks..."

go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
golangci-lint run --fix
