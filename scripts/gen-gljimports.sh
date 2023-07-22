#!/bin/bash

IMPORTS=$(go run ./cmd/gen-import-interop/main.go)
echo "$IMPORTS" > $1
