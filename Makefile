SHELL := bash

CLOJURE_STDLIB_VERSION := clojure-1.12.1
STDLIB_ORIGINALS_DIR := scripts/rewrite-core/originals
STDLIB_ORIGINALS := $(shell find $(STDLIB_ORIGINALS_DIR) -name '*.clj')
STDLIB := $(STDLIB_ORIGINALS:scripts/rewrite-core/originals/%=%)
STDLIB_ORIGINALS := $(addprefix scripts/rewrite-core/originals/,$(STDLIB))
STDLIB_TARGETS := $(addprefix pkg/stdlib/clojure/,$(STDLIB:.clj=.glj))

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
OA := $(OA-$(OS-ARCH))
GLJ := bin/$(OA)/glj
endif
endif

TEST_FILES := $(shell find ./test -name '*.glj' | sort)
TEST_TARGETS := $(addsuffix .test,$(TEST_FILES))

GOPLATFORMS := darwin_arm64 darwin_amd64 linux_arm64 linux_amd64 windows_amd64 windows_arm js_wasm
GLJIMPORTS=$(foreach platform,$(GOPLATFORMS),pkg/gen/gljimports/gljimports_$(platform).go)
# wasm should have .wasm suffix; others should not
BINS=$(foreach platform,$(GOPLATFORMS),bin/$(platform)/glj$(if $(findstring wasm,$(platform)),.wasm,))

# eventually, support multiple minor versions
GO_VERSION := 1.19.3
GO_CMD := go$(GO_VERSION)

.PHONY: all
all: gocmd $(STDLIB_TARGETS) go-generate aot $(GLJIMPORTS) $(BINS)

.PHONY: gocmd
gocmd:
	@$(GO_CMD) version 2>&1 > /dev/null || \
		(go install "golang.org/dl/$(GO_CMD)@latest" && \
		$(GO_CMD) download > /dev/null && $(GO_CMD) version > /dev/null)

.PHONY: go-generate
generate:
	@go generate ./...

.PHONY: aot
aot: gocmd $(STDLIB_TARGETS)
	@echo "(map compile '[clojure.core clojure.core.async clojure.string clojure.template clojure.test clojure.uuid clojure.walk glojure.go.types glojure.go.io])" | \
		GLOJURE_USE_AOT=false GLOJURE_STDLIB_PATH=./pkg/stdlib $(GO_CMD) run -tags glj_no_aot_stdlib ./cmd/glj

.PHONY: build
build: $(GLJ)

.PHONY: clean
clean:
	$(RM) report.html
	$(RM) -r bin/

pkg/gen/gljimports/gljimports_%.go: ./scripts/gen-gljimports.sh ./cmd/gen-import-interop/main.go ./internal/genpkg/genpkg.go \
					$(wildcard ./pkg/lang/*.go) $(wildcard ./pkg/runtime/*.go)
	@echo "Generating $@"
	@./scripts/gen-gljimports.sh $@ $* $(GO_CMD)

pkg/stdlib/clojure/%.glj: scripts/rewrite-core/originals/%.clj scripts/rewrite-core/run.sh scripts/rewrite-core/rewrite.clj
	@echo "Rewriting $< to $@"
	@mkdir -p $(dir $@)
	@scripts/rewrite-core/run.sh $< > $@

bin/%/glj: generate $(wildcard ./cmd/glj/*.go) $(wildcard ./pkg/**/*.go) $(wildcard ./internal/**/*.go)
	@echo "Building $@"
	@mkdir -p $(dir $@)
	@scripts/build-glj.sh $@ $*

bin/%/glj.wasm: $(wildcard ./cmd/glj/*.go) $(wildcard ./pkg/**/*.go) $(wildcard ./internal/**/*.go)
	@echo "Building $@"
	@mkdir -p $(dir $@)
	@scripts/build-glj.sh $@ $*

.PHONY: vet
vet:
	@go vet ./...

.PHONY: $(TEST_TARGETS)
$(TEST_TARGETS): gocmd $(GLJ)
	@$(GLJ) $(basename $@)

.PHONY: test
test: $(TEST_TARGETS) # vet - vet is disabled until we fix errors in generated code
	@cd test/clojure-test-suite && \
	../../$(GLJ) test-glojure.glj --expect-failures 38 --expect-errors 151 2>/dev/null

.PHONY: format
format:
	@if go fmt ./... | grep -q ''; then \
		echo "Files were formatted. Please commit the changes."; \
		exit 1; \
	fi

.PHONY: update-clojure-sources
update-clojure-sources:
	@scripts/rewrite-core/update-clojure-sources.sh $(CLOJURE_STDLIB_VERSION)
