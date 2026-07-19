#!/bin/bash

set -e

OUTPUT_FILE=$1
PLATFORM=$2
GO=$3

EXPECTED_GO_VERSION=$(awk '$1 == "go" {
    split($2, version, ".")
    print "go" version[1] "." version[2]
    exit
}' go.mod)
ACTUAL_GO_VERSION=$("$GO" env GOVERSION)
case "$ACTUAL_GO_VERSION" in
    "$EXPECTED_GO_VERSION".*) ;;
    *)
        echo "gljimports must be generated with ${EXPECTED_GO_VERSION}.x; got ${ACTUAL_GO_VERSION}" >&2
        exit 1
        ;;
esac

# make a temp directory with mktemp and build the project there
DIR=$(mktemp -d)
EXE="${DIR}/gen-import-interop"

"$GO" build -o "${EXE}" ./cmd/gen-import-interop

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
