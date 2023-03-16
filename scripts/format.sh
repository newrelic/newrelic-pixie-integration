#!/usr/bin/env bash

echo "[format] formatting code"

go install mvdan.cc/gofumpt@latest
gofumpt -l -w .
