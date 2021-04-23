#!/usr/bin/env bash

echo "[build-container] duilding container...."

docker build -t $1:$2 .