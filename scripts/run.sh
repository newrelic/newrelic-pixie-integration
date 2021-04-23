#!/usr/bin/env bash

echo "[run] running pixie-integration..."

source $1
export $(cut -d= -f1 $1)

go run cmd/main.go