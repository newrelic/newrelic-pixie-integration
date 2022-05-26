#!/usr/bin/env bash

echo "[build-container] building container...."

docker build -t $2:$3 --build-arg GOLANG_VERSION=$1 .
