# Usage:
#   make clean all test GO-VERSION=1.25.1

SHELL := bash

GO-VERSION ?= 1.19.3
CLOJURE-VERSION ?= 1.12.1

CLOJURE-STDLIB-VERSION := clojure-$(CLOJURE-VERSION)
STDLIB-ORIGINALS-DIR := scripts/rewrite-core/originals
STDLIB-ORIGINALS := $(shell find $(STDLIB-ORIGINALS-DIR) -name '*.clj')
STDLIB-NAMES := $(STDLIB-ORIGINALS:scripts/rewrite-core/originals/%=%)
STDLIB-ORIGINALS := $(STDLIB-NAMES:%=scripts/rewrite-core/originals/,%)
STDLIB-TARGETS := $(addprefix pkg/stdlib/clojure/,$(STDLIB-NAMES:.clj=.glj))

AOT-NAMESPACES := \
	clojure.core \
	clojure.core.async \
	clojure.string \
	clojure.template \
	clojure.test \
	clojure.walk \
	clojure.uuid \
	glojure.go.io \
	glojure.go.types \

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

TEST-FILES := $(shell find ./test -name '*.glj' | sort)
TEST-TARGETS := $(addsuffix .test,$(TEST-FILES))

GO-PLATFORMS := \
	darwin_arm64 \
	darwin_amd64 \
	linux_arm64 \
	linux_amd64 \
	windows_arm \
	windows_amd64 \
	js_wasm \

GLJ-IMPORTS=$(foreach platform,$(GO-PLATFORMS) \
              ,pkg/gen/gljimports/gljimports_$(platform).go)

# wasm should have .wasm suffix; others should not
GLJ-BINS=$(foreach platform,$(GO-PLATFORMS) \
	   ,bin/$(platform)/glj$(if $(findstring wasm,$(platform)),.wasm,))

GO-CMD := go$(GO-VERSION)

all: gocmd stdlib-targets generate aot glj-imports glj-bins

gocmd:
	@$(GO-CMD) version 2>&1 > /dev/null || \
		(go install "golang.org/dl/$(GO-CMD)@latest" && \
		$(GO-CMD) download > /dev/null && \
		$(GO-CMD) version > /dev/null)

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
	$(RM) -r bin/

pkg/gen/gljimports/gljimports_%.go: \
		./scripts/gen-gljimports.sh \
		./cmd/gen-import-interop/main.go \
		./internal/genpkg/genpkg.go \
		$(wildcard ./pkg/lang/*.go) \
		$(wildcard ./pkg/runtime/*.go)
	@echo "Generating $@"
	./scripts/gen-gljimports.sh $@ $* $(GO-CMD)

pkg/stdlib/clojure/%.glj: \
		scripts/rewrite-core/originals/%.clj \
		scripts/rewrite-core/run.sh \
		scripts/rewrite-core/rewrite.clj
	@echo "Rewriting $< to $@"
	@mkdir -p $(dir $@)
	scripts/rewrite-core/run.sh $< > $@

bin/%/glj: generate \
		$(wildcard ./cmd/glj/*.go) \
		$(wildcard ./pkg/**/*.go) \
		$(wildcard ./internal/**/*.go)
	@echo "Building $@"
	@mkdir -p $(dir $@)
	scripts/build-glj.sh $@ $*

bin/%/glj.wasm: \
		$(wildcard ./cmd/glj/*.go) \
		$(wildcard ./pkg/**/*.go) \
		$(wildcard ./internal/**/*.go)
	@echo "Building $@"
	@mkdir -p $(dir $@)
	scripts/build-glj.sh $@ $*

vet:
	go vet ./...

$(TEST-TARGETS): gocmd $(GLJ-CMD)
	$(GLJ-CMD) $(basename $@)

.PHONY: test
# vet - vet is disabled until we fix errors in generated code
test: $(TEST-TARGETS)
	cd test/clojure-test-suite && \
		../../$(GLJ-CMD) test-glojure.glj \
			--expect-failures 38 \
			--expect-errors 151 \
			2>/dev/null

format:
	@if go fmt ./... | grep -q ''; then \
		echo "Files were formatted. Please commit the changes."; \
		exit 1; \
	fi

update-clojure-sources:
	scripts/rewrite-core/update-clojure-sources.sh \
		$(CLOJURE-STDLIB-VERSION)
