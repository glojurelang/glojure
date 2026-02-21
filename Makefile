M := .cache/makes
$(shell [ -d $M ] || git clone -q https://github.com/makeplus/makes $M)

include $M/init.mk

GO-VERSION ?= 1.24.0
CLOJURE-VERSION ?= 1.12.1

include $M/go.mk
include $M/gh.mk
include $M/clean.mk
include $M/shell.mk

MAKES-CLEAN := \
  report.html \
  bin/ \
  scripts/rewrite-core/.cpcache/ \

MAKES-DISTCLEAN += \
  dist/ \
  .clj-kondo/ \
  .lsp/ \
  .vscode/

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
	,macos,$(if $(findstring linux,$(OS-TYPE))\
	,linux,$(if $(findstring freebsd,$(OS-TYPE))\
	,freebsd,$(if $(findstring netbsd,$(OS-TYPE))\
	,netbsd,$(if $(findstring openbsd,$(OS-TYPE))\
	,openbsd,$(if $(findstring dragonfly,$(OS-TYPE))\
	,dragonfly,))))))
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
OA-freebsd-arm64 := freebsd_arm64
OA-freebsd-int64 := freebsd_amd64
OA-openbsd-arm64 := openbsd_arm64
OA-openbsd-int64 := openbsd_amd64
OA-netbsd-arm64 := netbsd_arm64
OA-netbsd-int64 := netbsd_amd64
OA-dragonfly-int64 := dragonfly_amd64
GLJ-CMD := bin/$(OA-$(OS-ARCH))/glj
endif
endif

TEST-GLJ-DIR := test/glojure
TEST-GLJ-FILES := $(shell find $(TEST-GLJ-DIR) -name '*.glj' | sort)
TEST-GLJ-TARGETS := $(addsuffix .test,$(TEST-GLJ-FILES))
TEST-SUITE-DIR := test/clojure-test-suite
TEST-SUITE-FILE := test-glojure.glj

GO-PLATFORMS := \
	darwin_arm64 \
	darwin_amd64 \
	linux_arm64 \
	linux_amd64 \
	linux_arm \
	linux_riscv64 \
	linux_ppc64le \
	linux_s390x \
	linux_386 \
	windows_arm64 \
	windows_arm \
	windows_amd64 \
	windows_386 \
	freebsd_arm64 \
	freebsd_amd64 \
	freebsd_386 \
	openbsd_arm64 \
	openbsd_amd64 \
	netbsd_arm64 \
	netbsd_amd64 \
	dragonfly_amd64 \
	js_wasm \
	wasip1_wasm \
	$(EXTRA-GO-PLATFORMS)

# Disabled: solaris_amd64 (syscall.Syscall6 cross-compilation issue)
# Disabled: illumos_amd64 (syscall.Syscall6 cross-compilation issue)

GLJ-IMPORTS=$(foreach platform,$(GO-PLATFORMS) \
              ,pkg/gen/gljimports/gljimports_$(platform).go)

# wasm should have .wasm suffix; others should not
GLJ-BINS=$(foreach platform,$(GO-PLATFORMS) \
	   ,bin/$(platform)/glj$(if $(findstring wasm,$(platform)),.wasm,))

ALL-TARGETS := \
	$(if $(force),update-clojure-sources) \
	stdlib-targets \
	generate \
	aot \
	glj-imports \
	glj-bins \

#-------------------------------------------------------------------------------
# Dummy target for commands like:
#   make all force=1
#   make stdlib-targets force=1
force:

all: $(ALL-TARGETS)

stdlib-targets: $(STDLIB-TARGETS)

generate: $(GO)
	go generate ./...

aot: $(GO) $(STDLIB-TARGETS)
	GLOJURE_USE_AOT=false \
	GLOJURE_STDLIB_PATH=./pkg/stdlib \
	go run -tags glj_no_aot_stdlib ./cmd/glj \
	<<<"(map compile '[$(AOT-NAMESPACES)])"

glj-imports: $(GLJ-IMPORTS)

glj-bins: $(GLJ-BINS)

build: $(GLJ-CMD)

pkg/gen/gljimports/gljimports_%.go: \
	  ./scripts/gen-gljimports.sh \
	  ./cmd/gen-import-interop/main.go \
	  ./internal/genpkg/genpkg.go \
	  $(wildcard ./pkg/lang/*.go) \
	  $(wildcard ./pkg/runtime/*.go) \
	  $(if $(force),force)
	@echo "Generating $@"
	./scripts/gen-gljimports.sh $@ $* go

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

vet: $(GO)
	go vet ./...

.PHONY: test
# vet is disabled until we fix errors in generated code
test: test-glj test-suite  # vet

test-glj: $(TEST-GLJ-TARGETS)

test-suite: $(GLJ-CMD)
ifneq (,$(wildcard $(TEST-SUITE-DIR)))
	cd $(TEST-SUITE-DIR) && \
	  $(abspath $<) $(TEST-SUITE-FILE) \
	    --expect-failures 38 \
	    --expect-errors 151 \
	    2>/dev/null
endif

$(TEST-GLJ-TARGETS): $(GLJ-CMD)
	$< $(basename $@)

format: $(GO)
	@if go fmt ./... | grep -q ''; then \
	  echo "Files were formatted. Please commit the changes."; \
	  exit 1; \
	fi

update-clojure-sources:
	scripts/rewrite-core/update-clojure-sources.sh \
	  $(CLOJURE-STDLIB-VERSION)

RELEASE-PLATFORMS := linux_amd64 darwin_arm64

RELEASE-BINS := $(foreach p,$(RELEASE-PLATFORMS),bin/$(p)/glj)

release-dist:
	@$(if $(filter command line,$(origin VERSION)),,\
	  $(error VERSION is required on the command line))
	$(eval RELEASE_VER := $(patsubst v%,%,$(VERSION)))
	$(MAKE) stdlib-targets generate aot glj-imports $(RELEASE-BINS)
	mkdir -p dist
	$(foreach p,$(RELEASE-PLATFORMS), \
	  tar -czf dist/glj-$(RELEASE_VER)-$(p).tar.gz -C bin/$(p) glj ;)

remote ?= origin
git-push:
	$(eval HTTPS-URL := $(shell git remote get-url $(remote)))
	$(eval SSH-URL := $(subst https://github.com/,git@github.com:,$(HTTPS-URL)))
	git push $(SSH-URL) $(shell git rev-parse --abbrev-ref HEAD)

release: release-dist $(GH)
	$(eval RELEASE_VER := $(patsubst v%,%,$(VERSION)))
	git tag -a v$(RELEASE_VER) -m "Release v$(RELEASE_VER)"
	git push origin gloat
	git push origin v$(RELEASE_VER)
	$(GH-CMD) release create v$(RELEASE_VER) \
	  --repo gloathub/glojure \
	  --title "v$(RELEASE_VER)" \
	  --generate-notes \
	  dist/glj-$(RELEASE_VER)-*.tar.gz
