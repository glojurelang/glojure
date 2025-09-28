# Usage:
#   make clean all test GO-VERSION=1.25.1

SHELL := bash

GO-VERSION ?= 1.19.3
CLOJURE-VERSION ?= 1.12.1

CLOJURE-STDLIB-VERSION := clojure-$(CLOJURE-VERSION)
STDLIB-ORIGINALS-DIR := scripts/rewrite-core/originals
STDLIB-ORIGINALS := $(wildcard $(STDLIB-ORIGINALS-DIR)/*.clj)
STDLIB-NAMES := $(STDLIB-ORIGINALS:scripts/rewrite-core/originals/%=%)
STDLIB-ORIGINALS := $(STDLIB-NAMES:%=scripts/rewrite-core/originals/,%)
STDLIB-TARGETS := $(addprefix pkg/stdlib/clojure/,$(STDLIB-NAMES:.clj=.glj))

AOT-NAMESPACES := \
	clojure.core \
	clojure.core.async \
	clojure.string \
	clojure.template \
	clojure.test \
	clojure.uuid \
	clojure.walk \
	glojure.go.io \
	glojure.go.types \
	$(EXTRA-AOT-NAMESPACES)

OS-TYPE := $(shell bash -c 'echo $$OSTYPE')
OS-NAME := \
  $(if $(findstring darwin,$(OS-TYPE))\
	,macos,$(if $(findstring linux,$(OS-TYPE)),linux,))
ARCH-TYPE := $(shell bash -c 'echo $$MACHTYPE')
ARCH-NAME := \
  $(if $(or $(findstring arm64,$(ARCH-TYPE)),\
	          $(findstring aarch64,$(ARCH-TYPE)))\
	,arm64,$(if $(findstring x86_64,$(ARCH-TYPE)),int64,))

ifdef OS-NAME
ifdef ARCH-NAME
OS-ARCH := $(OS-NAME)-$(ARCH-NAME)
OA-linux-arm64 := linux_arm64
OA-linux-int64 := linux_amd64
OA-macos-arm64 := darwin_arm64
OA-macos-int64 := darwin_amd64
GLJ-CMD := bin/$(OA-$(OS-ARCH))/glj
endif
endif

TEST-GLJ-DIR := test/glojure
TEST-SUITE-DIR := test/clojure-test-suite
TEST-SUITE-FILE := test-glojure.glj

GO-PLATFORMS := \
	darwin_arm64 \
	darwin_amd64 \
	linux_arm64 \
	linux_amd64 \
	windows_arm \
	windows_amd64 \
	js_wasm \
	$(EXTRA-GO-PLATFORMS)

GLJ-IMPORTS=$(foreach platform,$(GO-PLATFORMS) \
              ,pkg/gen/gljimports/gljimports_$(platform).go)

# wasm should have .wasm suffix; others should not
GLJ-BINS=$(foreach platform,$(GO-PLATFORMS) \
	   ,bin/$(platform)/glj$(if $(findstring wasm,$(platform)),.wasm,))

# TEST-RUNNER-BIN no longer needed - using glj -m instead

GO-CMD := go$(GO-VERSION)

ALL-TARGETS := \
	$(if $(force),update-clojure-sources) \
	stdlib-targets \
	generate \
	aot \
	glj-imports \
	glj-bins \

#-------------------------------------------------------------------------------
default: all

.PHONY: help
help:
	@echo "Glojure Makefile targets:"
	@echo ""
	@echo "Building:"
	@echo "  all                - Build everything"
	@echo "  build              - Build glj for current platform"
	@echo "  clean              - Remove built files"
	@echo ""
	@echo "Testing with new test runner:"
	@echo "  test               - Run all tests (test-glj and test-suite)"
	@echo "  test-glj           - Run tests in test/glojure/test_glojure"
	@echo "  test-verbose       - Run tests with verbose output"
	@echo "  test-ns NS=pattern - Run tests matching namespace pattern"
	@echo "  test-tap           - Run tests with TAP format output"
	@echo "  test-json          - Run tests with JSON format output"
	@echo "  test-list          - List all test namespaces"
	@echo ""
	@echo "Examples:"
	@echo "  make test-ns NS=string   - Run string tests"
	@echo "  make test-ns NS=basic    - Run basic tests"
	@echo "  make test-json           - Get JSON test results"

# Dummy target for commands like:
#   make all force=1
#   make stdlib-targets force=1
force:

all: $(ALL-TARGETS)

gocmd:
	@$(GO-CMD) version &> /dev/null || { \
		(go install "golang.org/dl/$(GO-CMD)@latest" && \
		$(GO-CMD) download > /dev/null && \
		$(GO-CMD) version > /dev/null); }

stdlib-targets: $(STDLIB-TARGETS)

generate:
	go generate ./...

aot: gocmd $(STDLIB-TARGETS)
	GLOJURE_USE_AOT=false \
	GLOJURE_STDLIB_PATH=./pkg/stdlib \
	$(GO-CMD) run -tags glj_no_aot_stdlib ./cmd/glj \
	<<<"(map compile '[$(AOT-NAMESPACES)])"

glj-imports: $(GLJ-IMPORTS)

glj-bins: $(GLJ-BINS)

build: $(GLJ-CMD)

clean:
	$(RM) report.html
	$(RM) -r bin/ scripts/rewrite-core/.cpcache/

pkg/gen/gljimports/gljimports_%.go: \
		./scripts/gen-gljimports.sh \
		./cmd/gen-import-interop/main.go \
		./internal/genpkg/genpkg.go \
		$(wildcard ./pkg/lang/*.go) \
		$(wildcard ./pkg/runtime/*.go) \
		$(if $(force),force)
	@echo "Generating $@"
	./scripts/gen-gljimports.sh $@ $* $(GO-CMD)

pkg/stdlib/clojure/%.glj: \
		scripts/rewrite-core/originals/%.clj \
		scripts/rewrite-core/run.sh \
		scripts/rewrite-core/rewrite.clj \
		$(if $(force),force)
	@echo "Rewriting $< to $@"
	@mkdir -p $(dir $@)
	scripts/rewrite-core/run.sh $< > $@

bin/%/glj: generate \
		$(wildcard ./cmd/glj/*.go) \
		$(wildcard ./pkg/**/*.go) \
		$(wildcard ./internal/**/*.go) \
		$(if $(force),force)
	@echo "Building $@"
	@mkdir -p $(dir $@)
	scripts/build-glj.sh $@ $*

bin/%/glj.wasm: \
		$(wildcard ./cmd/glj/*.go) \
		$(wildcard ./pkg/**/*.go) \
		$(wildcard ./internal/**/*.go) \
		$(if $(force),force)
	@echo "Building $@"
	@mkdir -p $(dir $@)
	scripts/build-glj.sh $@ $*

vet:
	go vet ./...

.PHONY: test test-glj test-verbose test-ns test-tap test-json test-list
# vet is disabled until we fix errors in generated code
test: test-glj test-suite  # vet

# Run tests in test/glojure/test_glojure using the new test runner
test-glj: $(GLJ-CMD)
	$(GLJ-CMD) -m glojure.test-runner --dir test/glojure --format console

test-suite: $(GLJ-CMD)
	cd $(TEST-SUITE-DIR) && \
		$(abspath $<) $(TEST-SUITE-FILE) \
			--expect-failures 38 \
			--expect-errors 151 \
			2>/dev/null

# Test runner now integrated into glj with -m flag

# Run all tests with verbose output
test-verbose: $(GLJ-CMD)
	$(GLJ-CMD) -m glojure.test-runner --dir test/glojure --verbose

# Run specific test namespace with test runner
test-ns: $(GLJ-CMD)
	@if [ -z "$(NS)" ]; then \
		echo "Usage: make test-ns NS=pattern"; \
		echo "Example: make test-ns NS=basic"; \
		exit 1; \
	fi
	$(GLJ-CMD) -m glojure.test-runner --dir test/glojure --namespace ".*$(NS).*" --verbose

# Run tests with TAP format output (useful for CI)
test-tap: $(GLJ-CMD)
	$(GLJ-CMD) -m glojure.test-runner --dir test/glojure --format tap

# Run tests with JSON format output
test-json: $(GLJ-CMD)
	$(GLJ-CMD) -m glojure.test-runner --dir test/glojure --format json

# List all test namespaces without running them
test-list: $(GLJ-CMD)
	$(GLJ-CMD) -m glojure.test-runner --dir test/glojure --list

# Old per-file test targets removed - using test runner instead

format:
	@if go fmt ./... | grep -q ''; then \
		echo "Files were formatted. Please commit the changes."; \
		exit 1; \
	fi

update-clojure-sources:
	scripts/rewrite-core/update-clojure-sources.sh \
		$(CLOJURE-STDLIB-VERSION)
