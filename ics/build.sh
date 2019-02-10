#!/bin/bash

set -e

export arch=$(uname -m)

if [ "$arch" = "armv7l" ] ; then
    echo "Build not supported on $arch, use cross-build."
    exit 1
fi

cd ..
GIT_COMMIT=$(git rev-list -1 HEAD)
VERSION=$(git describe --all --exact-match `git rev-parse HEAD` | grep tags | sed 's/tags\///')
cd ics

if [ ! $http_proxy == "" ]
then
    docker build --no-cache --build-arg https_proxy=$https_proxy --build-arg http_proxy=$http_proxy \
        --build-arg GIT_COMMIT=$GIT_COMMIT --build-arg VERSION=$VERSION -t zhangjyr/hyperfaas-ics:build .
else
    docker build --no-cache --build-arg VERSION=$VERSION --build-arg GIT_COMMIT=$GIT_COMMIT -t zhangjyr/hyperfaas-ics:build .
fi

docker create --name buildoutput zhangjyr/hyperfaas-ics:build echo

docker cp buildoutput:/go/src/github.com/openfaas/faas/ics/ics ./ics
docker cp buildoutput:/go/src/github.com/openfaas/faas/ics/ics-armhf ./ics-armhf
docker cp buildoutput:/go/src/github.com/openfaas/faas/ics/ics-arm64 ./ics-arm64
docker cp buildoutput:/go/src/github.com/openfaas/faas/ics/ics.exe ./ics.exe

docker rm buildoutput
