M := $(or $(MAKES_REPO_DIR),.cache/makes)
C := bc32749a8dcc250e375b3a6d3572c3260a972efb
$(shell [ -d $M ] || git clone -q https://github.com/makeplus/makes $M)
$(shell [ -d $M ] || ( \
  git clone -depth=1 -q https://github.com/makeplus/makes $M && \
  git -C $M reset -q --hard $C))
include $M/init.mk
MAKES-NO-RULES := true
GO-VERSION ?= 1.19.3
GO-VERSION ?= 1.22.0
include $M/go.mk
include $M/clojure.mk
include $M/clean.mk
include $M/shell.mk

MAKES-CLEAN := bin/
MAKES-DISTCLEAN := .cache/ .clj-kondo/ .lsp/ .vscode/

STDLIB_ORIGINALS_DIR := scripts/rewrite-core/originals
STDLIB_ORIGINALS := $(shell find $(STDLIB_ORIGINALS_DIR) -name '*.clj')
STDLIB := $(STDLIB_ORIGINALS:scripts/rewrite-core/originals/%=%)
STDLIB_ORIGINALS := $(addprefix scripts/rewrite-core/originals/,$(STDLIB))
STDLIB_TARGETS := $(addprefix pkg/stdlib/glojure/,$(STDLIB:.clj=.glj))

TEST_FILES := $(shell find ./test -name '*.glj')
TEST_TARGETS := $(addsuffix .test,$(TEST_FILES))

GOPLATFORMS := \
  darwin_arm64 \
  darwin_amd64 \
  linux_arm64 \
  linux_amd64 \
  windows_amd64 \
  windows_arm \
  js_wasm \

# Set PATH so that glj is in the PATH after 'make shell'
override PATH := $(subst $(space),:,$(GOPLATFORMS:%=bin/%)):$(PATH)

GLJIMPORTS=$(foreach platform,$(GOPLATFORMS),pkg/gen/gljimports/gljimports_$(platform).go)
# wasm should have .wasm suffix; others should not
BINS=$(foreach platform,$(GOPLATFORMS),bin/$(platform)/glj$(if $(findstring wasm,$(platform)),.wasm,))

GLJ-DEPS := \
  $(wildcard ./cmd/glj/*.go) \
  $(wildcard ./pkg/**/*.go) \
  $(wildcard ./internal/**/*.go) \

all: $(STDLIB_TARGETS) generate $(GLJIMPORTS) $(BINS)

OA-linux-arm64 := linux_arm64
OA-linux-int64 := linux_amd64
OA-macos-arm64 := darwin_arm64
OA-macos-int64 := darwin_amd64
OA := $(OA-$(OS-ARCH))

ifdef OA
build: bin/$(OA)/glj
endif

generate: $(GO)
	@go generate ./...

pkg/gen/gljimports/gljimports_%.go: $(GO) \
	  ./scripts/gen-gljimports.sh \
	  ./cmd/gen-import-interop/main.go \
	  ./internal/genpkg/genpkg.go \
	  $(wildcard ./pkg/lang/*.go) \
	  $(wildcard ./pkg/runtime/*.go)
	@echo "Generating $@"
	@./scripts/gen-gljimports.sh $@ $* go

pkg/stdlib/glojure/%.glj: \
	  scripts/rewrite-core/originals/%.clj \
	  scripts/rewrite-core/run.sh \
	  scripts/rewrite-core/rewrite.clj
	@echo "Rewriting $< to $@"
	@mkdir -p $(dir $@)
	@scripts/rewrite-core/run.sh $< > $@

bin/%/glj: $(GLJ-DEPS)
	@echo "Building $@"
	@mkdir -p $(dir $@)
	@scripts/build-glj.sh $@ $*

bin/%/glj.wasm: $(GLJ-DEPS)
	@echo "Building $@"
	@mkdir -p $(dir $@)
	@scripts/build-glj.sh $@ $*

vet: $(GO)
	@go vet ./...

$(TEST_TARGETS):
	go run ./cmd/glj/main.go $(basename $@)

.PHONY: test
test: vet $(TEST_TARGETS)
