#!/usr/bin/env bash

echo "[build-container] building container...."

docker build -t $1:$2 --build-arg image_version=$2 .
