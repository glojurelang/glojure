SHELL := bash

CLOJURE-STDLIB-VERSION := clojure-1.12.1
STDLIB-ORIGINALS-DIR := scripts/rewrite-core/originals
STDLIB-ORIGINALS := $(shell find $(STDLIB-ORIGINALS-DIR) -name '*.clj')
STDLIB-NAMES := $(STDLIB-ORIGINALS:scripts/rewrite-core/originals/%=%)
STDLIB-ORIGINALS := $(addprefix scripts/rewrite-core/originals/,$(STDLIB-NAMES))
STDLIB-TARGETS := $(addprefix pkg/stdlib/clojure/,$(STDLIB-NAMES:.clj=.glj))

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

GO-PLATFORMS := darwin_arm64 darwin_amd64 linux_arm64 linux_amd64 windows_amd64 windows_arm js_wasm
GLJ-IMPORTS=$(foreach platform,$(GO-PLATFORMS),pkg/gen/gljimports/gljimports_$(platform).go)
# wasm should have .wasm suffix; others should not
GLJ-BINS=$(foreach platform,$(GO-PLATFORMS),bin/$(platform)/glj$(if $(findstring wasm,$(platform)),.wasm,))

# eventually, support multiple minor versions
GO-VERSION := 1.19.3
GO-CMD := go$(GO-VERSION)

.PHONY: all
all: gocmd $(STDLIB-TARGETS) go-generate aot $(GLJ-IMPORTS) $(GLJ-BINS)

.PHONY: gocmd
gocmd:
	@$(GO-CMD) version 2>&1 > /dev/null || \
		(go install "golang.org/dl/$(GO-CMD)@latest" && \
		$(GO-CMD) download > /dev/null && $(GO-CMD) version > /dev/null)

.PHONY: go-generate
generate:
	@go generate ./...

.PHONY: aot
aot: gocmd $(STDLIB-TARGETS)
	@echo "(map compile '[clojure.core clojure.core.async clojure.string clojure.template clojure.test clojure.uuid clojure.walk glojure.go.types glojure.go.io])" | \
		GLOJURE_USE_AOT=false GLOJURE_STDLIB_PATH=./pkg/stdlib $(GO-CMD) run -tags glj_no_aot_stdlib ./cmd/glj

.PHONY: build
build: $(GLJ-CMD)

.PHONY: clean
clean:
	$(RM) report.html
	$(RM) -r bin/

pkg/gen/gljimports/gljimports_%.go: ./scripts/gen-gljimports.sh ./cmd/gen-import-interop/main.go ./internal/genpkg/genpkg.go \
					$(wildcard ./pkg/lang/*.go) $(wildcard ./pkg/runtime/*.go)
	@echo "Generating $@"
	@./scripts/gen-gljimports.sh $@ $* $(GO-CMD)

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

.PHONY: $(TEST-TARGETS)
$(TEST-TARGETS): gocmd $(GLJ-CMD)
	@$(GLJ-CMD) $(basename $@)

.PHONY: test
test: $(TEST-TARGETS) # vet - vet is disabled until we fix errors in generated code
	@cd test/clojure-test-suite && \
	../../$(GLJ-CMD) test-glojure.glj --expect-failures 38 --expect-errors 151 2>/dev/null

.PHONY: format
format:
	@if go fmt ./... | grep -q ''; then \
		echo "Files were formatted. Please commit the changes."; \
		exit 1; \
	fi

.PHONY: update-clojure-sources
update-clojure-sources:
	@scripts/rewrite-core/update-clojure-sources.sh $(CLOJURE-STDLIB-VERSION)
