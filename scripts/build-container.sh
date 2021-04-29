#!/usr/bin/env bash

echo "[build-container] duilding container...."

docker build -t $2:$3 --build-arg GOLANG_VERSION=$1 .