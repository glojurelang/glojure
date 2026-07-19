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

LDFLAGS="-s -w"
if [ -n "$GLJ_VERSION" ]; then
    LDFLAGS="$LDFLAGS -X github.com/glojurelang/glojure/pkg/runtime.version=$GLJ_VERSION"
fi

# When GO_REPLACE is set, create a temporary go.work file with replace
# directives for local development.  Multiple replacements can be
# space-separated: "mod1=path1 mod2=path2".
gowork_cleanup() { rm -f go.work go.work.sum; }
if [ -n "$GO_REPLACE" ]; then
    gowork_cleanup
    go work init .
    for r in $GO_REPLACE; do
        go work edit -replace "$r"
    done
    trap gowork_cleanup EXIT
fi

GOOS=$OS GOARCH=$ARCH go build -trimpath -ldflags "$LDFLAGS" -o "$1" ./cmd/glj
