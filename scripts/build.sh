#!/usr/bin/env bash

echo "[build] building pixie-integration executable..."

go build -o bin/pixie-integration cmd/main.go