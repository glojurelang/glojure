#!/bin/bash

set -e

# make a temp directory with mktemp and build the project there
DIR=$(mktemp -d)
EXE="${DIR}/gen-import-interop"

go build -o "${EXE}" ./cmd/gen-import-interop

OUTPUT_FILE=$1
PLATFORM=$2
GO=$3

IFS='_' read -ra OS_ARCH <<< "$PLATFORM"

OS=${OS_ARCH[0]}
ARCH=${OS_ARCH[1]}

if [ "$ARCH" == "" ]; then
    BUILD_TAG="$OS"
else
    BUILD_TAG="$ARCH && $OS"
fi

# disable CGO to avoid cross-compilation issues on darwin.
IMPORTS=$(GOROOT=$($GO env GOROOT) CGO_ENABLED=0 GOOS=$OS GOARCH=$ARCH "$EXE")
echo "//go:build $BUILD_TAG" > "$OUTPUT_FILE"
echo >> "$OUTPUT_FILE"
echo "$IMPORTS" >> "$OUTPUT_FILE"

# clean up
rm -rf "${DIR}"
