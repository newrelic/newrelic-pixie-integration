#!/usr/bin/env bash

echo "[setup] configuring git hooks..."

chmod +x .githooks/*
cp .githooks/* .git/hooks/