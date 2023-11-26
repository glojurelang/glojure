
STDLIB_ORIGINALS_DIR := scripts/rewrite-core/originals
STDLIB_ORIGINALS := $(shell find $(STDLIB_ORIGINALS_DIR) -name '*.clj')
STDLIB := $(STDLIB_ORIGINALS:scripts/rewrite-core/originals/%=%)
STDLIB_ORIGINALS := $(addprefix scripts/rewrite-core/originals/,$(STDLIB))
STDLIB_TARGETS := $(addprefix pkg/stdlib/glojure/,$(STDLIB:.clj=.glj))

TEST_FILES := $(shell find ./test -name '*.glj')
TEST_TARGETS := $(addsuffix .test,$(TEST_FILES))

GOPLATFORMS := darwin_arm64 darwin_amd64 linux_arm64 linux_amd64 windows_amd64 windows_arm js_wasm
GLJIMPORTS=$(foreach platform,$(GOPLATFORMS),pkg/gen/gljimports/gljimports_$(platform).go)
# wasm should have .wasm suffix; others should not
BINS=$(foreach platform,$(GOPLATFORMS),bin/$(platform)/glj$(if $(findstring wasm,$(platform)),.wasm,))

# eventually, support multiple minor versions
GO_VERSION := 1.19.3
GENIMPORTS_GO_CMD := go$(GO_VERSION)

.PHONY: all
all: gocmd $(STDLIB_TARGETS) generate $(GLJIMPORTS) $(BINS)

.PHONY: gocmd
gocmd:
	@$(GENIMPORTS_GO_CMD) version 2>&1 > /dev/null || \
		(go install "golang.org/dl/$(GENIMPORTS_GO_CMD)@latest" && \
		$(GENIMPORTS_GO_CMD) download > /dev/null && $(GENIMPORTS_GO_CMD) version > /dev/null)

.PHONY: generate
generate:
	@go generate ./...

pkg/gen/gljimports/gljimports_%.go: ./scripts/gen-gljimports.sh ./cmd/gen-import-interop/main.go ./internal/genpkg/genpkg.go \
					$(wildcard ./pkg/lang/*.go) $(wildcard ./pkg/runtime/*.go)
	@echo "Generating $@"
	@./scripts/gen-gljimports.sh $@ $* $(GENIMPORTS_GO_CMD)

pkg/stdlib/glojure/%.glj: scripts/rewrite-core/originals/%.clj scripts/rewrite-core/run.sh scripts/rewrite-core/rewrite.clj
	@echo "Rewriting $< to $@"
	@mkdir -p $(dir $@)
	@scripts/rewrite-core/run.sh $< > $@

bin/%/glj: $(wildcard ./cmd/glj/*.go) $(wildcard ./pkg/**/*.go) $(wildcard ./internal/**/*.go)
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
$(TEST_TARGETS): gocmd
	@go run ./cmd/glj/main.go $(basename $@) | tee /dev/stderr | grep FAIL > /dev/null && exit 1 || exit 0

.PHONY: test
test: vet $(TEST_TARGETS)
	@go test ./...
