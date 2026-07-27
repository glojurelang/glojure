R := https://github.com/makeplus/makes
M := .cache/makes
$(shell [ -d '$M' ] || git clone -q $R '$M')

include $M/init.mk

GO-VERSION ?= 1.24.0
CLOJURE-CLI-VERSION ?= 1.12.4.1602
CLOJURE-STDLIB-SOURCE-VERSION ?= 1.12.4
CLOJURE-VERSION ?= $(CLOJURE-CLI-VERSION)

include $M/go.mk
include $M/perl.mk
CLOJURE-DEPS += $(PERL)
include $M/clojure.mk
include $M/gh.mk
include $M/clean.mk
include $M/yamlscript.mk
include $M/shell.mk

CLOJURE-STDLIB-VERSION := clojure-$(CLOJURE-STDLIB-SOURCE-VERSION)
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
TEST-SUITE-REPO := https://github.com/glojurelang/clojure-test-suite.git
TEST-SUITE-BRANCH := glojure
TEST-SUITE-DIR := test/clojure-test-suite
TEST-SUITE-FILE := test-glojure.glj
TEST-SUITE-EXPECT-FAILURES ?= 0
TEST-SUITE-EXPECT-ERRORS ?= 1
TEST-SUITE-EXPECT-LOAD-ERRORS ?= 7

MAKES-CLEAN := \
  report.html \
  bin/ \
  scripts/rewrite-core/.cpcache/ \
  $(TEST-SUITE-DIR) \

MAKES-REALCLEAN += \
  dist/ \
  .clj-kondo/ \
  .lsp/ \
  .vscode/

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
	plan9_amd64 \
	plan9_386 \
	plan9_arm \
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
	  $(if $(force),force) \
	  | $(CLOJURE)
	@echo "Rewriting $< to $@"
	@mkdir -p $(dir $@)
	tmp=$@.tmp; \
	  HOME="$(LOCAL-HOME)" CLJ="$(CLOJURE)" scripts/rewrite-core/run.sh $< > $$tmp && \
	  mv $$tmp $@

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

# vet is disabled until we fix errors in generated code
test: test-aot-runtime test-glj  # vet
	($(MAKE) test-suite v=1 || $(MAKE) test-suite v=1) || $(MAKE) test-suite v=1

TEST-COMPARE-LOAD := $(shell \
	processors=$$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 2); \
	awk -v processors="$$processors" \
	  'BEGIN { threshold = processors * 0.75; \
	           print (threshold < 1 ? 1 : threshold) }')

test-compare: $(YS)
	@scripts/make-test-compare \
	  '$(if $(with),$(with),origin/main)' '$(file)' \
	  '$(if $(load),$(load),$(TEST-COMPARE-LOAD))'

test-aot-runtime: $(GO)
	go test -tags glj_aot_runtime ./pkg/glj ./pkg/gljmain ./pkg/runtime

.PHONY: test-aot test-suite-aot
test-aot: test-aot-runtime test-glj
	$(MAKE) test-suite-aot

test-suite-aot: $(GO) $(STDLIB-TARGETS) generate aot $(TEST-SUITE-DIR)
	cd $(TEST-SUITE-DIR) && git checkout $(TEST-SUITE-BRANCH)
	TEST_SUITE_EXPECT_FAILURES=$(TEST-SUITE-EXPECT-FAILURES) \
	TEST_SUITE_EXPECT_ERRORS=$(TEST-SUITE-EXPECT-ERRORS) \
	TEST_SUITE_EXPECT_LOAD_ERRORS=$(TEST-SUITE-EXPECT-LOAD-ERRORS) \
	  scripts/test-suite-aot $(abspath $(TEST-SUITE-DIR))

test-glj: $(TEST-GLJ-TARGETS)

$(TEST-SUITE-DIR):
	git clone --branch $(TEST-SUITE-BRANCH) $(TEST-SUITE-REPO) $@

test-suite: $(GLJ-CMD) $(TEST-SUITE-DIR)
	cd $(TEST-SUITE-DIR) && git checkout $(TEST-SUITE-BRANCH)
	cd $(TEST-SUITE-DIR) && \
	  $(abspath $<) $(TEST-SUITE-FILE) \
	    $(if $(TEST-SUITE-EXPECT-FAILURES),--expect-failures $(TEST-SUITE-EXPECT-FAILURES)) \
	    $(if $(TEST-SUITE-EXPECT-ERRORS),--expect-errors $(TEST-SUITE-EXPECT-ERRORS)) \
	    --expect-load-errors $(TEST-SUITE-EXPECT-LOAD-ERRORS) \
	    $(if $(v),,2>/dev/null)

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

RELEASE-PLATFORMS := \
  linux_amd64 \
  linux_arm64 \
  darwin_arm64 \
  $(EXTRA-RELEASE-PLATFORMS)

RELEASE-BINS := $(foreach p,$(RELEASE-PLATFORMS),bin/$(p)/glj)

release-dist:
	@$(if $(filter command line,$(origin VERSION)),,\
	  $(error VERSION is required on the command line))
	$(eval RELEASE_VER := $(patsubst v%,%,$(VERSION)))
	GLJ_VERSION=v$(RELEASE_VER) $(MAKE) stdlib-targets generate aot glj-imports $(RELEASE-BINS)
	mkdir -p dist
	$(foreach p,$(RELEASE-PLATFORMS), \
	  tar -czf dist/glj-$(RELEASE_VER)-$(p).tar.gz -C bin/$(p) glj ;)
ifdef RELEASE-PLAN9-AMD64
	@echo "Building Plan 9/amd64 binary (nospinbitmutex)"
	mkdir -p bin/plan9_amd64
	CGO_ENABLED=0 GOOS=plan9 GOARCH=amd64 \
	  GOEXPERIMENT=nospinbitmutex \
	  go build -o bin/plan9_amd64/glj ./cmd/glj
	tar -czf dist/glj-$(RELEASE_VER)-plan9_amd64.tar.gz -C bin/plan9_amd64 glj
endif

remote ?= origin
release-branch ?= main
git-push:
	$(eval HTTPS-URL := $(shell git remote get-url $(remote)))
	$(eval SSH-URL := $(subst https://github.com/,git@github.com:,$(HTTPS-URL)))
	git push $(SSH-URL) $(shell git rev-parse --abbrev-ref HEAD)

release: $(GH)
	@$(if $(filter command line,$(origin VERSION)),,\
	  $(error VERSION is required on the command line))
	$(eval RELEASE_VER := $(patsubst v%,%,$(VERSION)))
	@echo "=== Release v$(RELEASE_VER) ==="
	$(MAKE) clean
	$(MAKE) stdlib-targets
	$(MAKE) generate aot
	$(MAKE) glj-imports force=1
	GLJ_VERSION=v$(RELEASE_VER) $(MAKE) build
	($(MAKE) test || $(MAKE) test) || $(MAKE) test
	git add -A
	git diff --cached --quiet || \
	  git commit -m "Builds for v$(RELEASE_VER)"
	$(MAKE) release-dist VERSION=$(VERSION)
	git tag -f -a v$(RELEASE_VER) -m "Release v$(RELEASE_VER)"
ifeq ($(release-branch),main)
	git push $(remote) HEAD:$(release-branch)
else
	git fetch $(remote) $(release-branch)
	git push --force-with-lease=refs/heads/$(release-branch) \
	  $(remote) HEAD:$(release-branch)
endif
	git push $(remote) v$(RELEASE_VER)
	$(GH) release create v$(RELEASE_VER) \
	  --repo glojurelang/glojure \
	  --title "v$(RELEASE_VER)" \
	  $(if $(findstring -rc,$(RELEASE_VER)),--prerelease,) \
	  --generate-notes \
	  dist/glj-$(RELEASE_VER)-*.tar.gz
	@echo "=== Release v$(RELEASE_VER) complete ==="
