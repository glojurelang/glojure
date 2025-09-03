M := $(or $(MAKES_REPO_DIR),.cache/makes)
C := ca8c2c25e66cf6bfcf8c993502de5b98da5beaf5
$(shell [ -d $M ] || git clone -q https://github.com/makeplus/makes $M)
$(shell [ -d $M ] || ( \
  git clone -depth=1 -q https://github.com/makeplus/makes $M && \
  git -C $M reset -q --hard $C))
include $M/init.mk
MAKES-NO-RULES := true
GO-VERSION ?= 1.25.0
include $M/go.mk
CLOJURE-VERSION := 1.12.1.1550
include $M/clojure.mk
include $M/clean.mk
include $M/shell.mk

MAKES-CLEAN := bin/
MAKES-DISTCLEAN := .cache/ .clj-kondo/ .lsp/ .vscode/

CLOJURE-STDLIB-VERSION := clojure-1.12.1
STDLIB-ORIGINALS-DIR := scripts/rewrite-core/originals
STDLIB-ORIGINALS := $(shell find $(STDLIB-ORIGINALS-DIR) -name '*.clj')
STDLIB := $(STDLIB-ORIGINALS:scripts/rewrite-core/originals/%=%)
STDLIB-ORIGINALS := $(addprefix scripts/rewrite-core/originals/,$(STDLIB))
STDLIB-TARGETS := $(addprefix pkg/stdlib/glojure/,$(STDLIB:.clj=.glj))

GOPLATFORMS := \
  darwin_arm64 \
  darwin_amd64 \
  linux_arm64 \
  linux_amd64 \
  windows_amd64 \
  windows_arm \
  js_wasm \
  wasip1_wasm \

# Set PATH so that glj is in the PATH after 'make shell'
override PATH := $(subst $(space),:,$(GOPLATFORMS:%=$(ROOT)/bin/%)):$(PATH)

GLJIMPORTS=$(foreach platform,$(GOPLATFORMS),pkg/gen/gljimports/gljimports_$(platform).go)
# wasm should have .wasm suffix; others should not
BINS=$(foreach platform,$(GOPLATFORMS),bin/$(platform)/glj$(if $(findstring wasm,$(platform)),.wasm,))

GLJ-DEPS := \
	$(GO) \
  $(wildcard ./cmd/glj/*.go) \
  $(wildcard ./pkg/**/*.go) \
  $(wildcard ./internal/**/*.go) \

all: $(STDLIB-TARGETS) generate $(GLJIMPORTS) $(BINS)

OA-linux-arm64 := linux_arm64
OA-linux-int64 := linux_amd64
OA-macos-arm64 := darwin_arm64
OA-macos-int64 := darwin_amd64
OA := $(OA-$(OS-ARCH))

GLJ := bin/$(OA)/glj

TEST-FILES := $(shell find ./test -name '*.glj' | sort)
TEST-TARGETS := $(addsuffix .test,$(TEST-FILES))


ifdef OA
build: $(GLJ)
endif

generate: $(GO)
	go generate ./...

pkg/gen/gljimports/gljimports_%.go: $(GO) \
	  ./scripts/gen-gljimports.sh \
	  ./cmd/gen-import-interop/main.go \
	  ./internal/genpkg/genpkg.go \
	  $(wildcard ./pkg/lang/*.go) \
	  $(wildcard ./pkg/runtime/*.go)
	@echo "Generating $@"
	./scripts/gen-gljimports.sh $@ $* go

pkg/stdlib/glojure/%.glj: $(CLOJURE) \
	  scripts/rewrite-core/originals/%.clj \
	  scripts/rewrite-core/run.sh \
	  scripts/rewrite-core/rewrite.clj
	@echo "Rewriting $< to $@"
	@mkdir -p $(dir $@)
	scripts/rewrite-core/run.sh $< > $@

bin/%/glj: $(GLJ-DEPS)
	@echo "Building $@"
	@mkdir -p $(dir $@)
	scripts/build-glj.sh $@ $*

bin/%/glj.wasm: $(GLJ-DEPS)
	@echo "Building $@"
	@mkdir -p $(dir $@)
	scripts/build-glj.sh $@ $*

vet: $(GO)
	go vet ./...

$(TEST-TARGETS): $(GO) $(GLJ)
	go run ./cmd/glj/main.go $(basename $@)

.PHONY: test
test: vet $(TEST-TARGETS)

format: $(GO)
	@if go fmt ./... | grep -q ''; then \
		echo "Files were formatted. Please commit the changes."; \
		exit 1; \
	fi

update-clojure-sources:
	scripts/rewrite-core/update-clojure-sources.sh $(CLOJURE-STDLIB-VERSION)

test-bug: $(GLJ)
	-glj <(echo '(throw (Exception. "foo"))')
	-glj <(echo '(throw (new Exception "foo"))')
