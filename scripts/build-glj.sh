#!/bin/bash

set -e

PLATFORM=$2

IFS='_' read -ra OS_ARCH <<< "$PLATFORM"

OS=${OS_ARCH[0]}
ARCH=${OS_ARCH[1]}

if [ "$ARCH" == "" ]; then
    BUILD_TAG="$OS"
else
    BUILD_TAG="$ARCH && $OS"
fi

GOOS=$OS GOARCH=$ARCH go build -o $1 ./cmd/glj
