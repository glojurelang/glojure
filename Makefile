
STDLIB_ORIGINALS_DIR := scripts/rewrite-core/originals
STDLIB_ORIGINALS := $(shell find $(STDLIB_ORIGINALS_DIR) -name '*.clj')
STDLIB := $(STDLIB_ORIGINALS:scripts/rewrite-core/originals/%=%)
STDLIB_ORIGINALS := $(addprefix scripts/rewrite-core/originals/,$(STDLIB))
STDLIB_TARGETS := $(addprefix pkg/stdlib/glojure/,$(STDLIB:.clj=.glj))

TEST_FILES := $(shell find ./test -name '*.glj')
TEST_TARGETS := $(addsuffix .test,$(TEST_FILES))

GOPLATFORMS := darwin_arm64 darwin_amd64 linux_arm64 linux_amd64 windows
GLJIMPORTS=$(foreach platform,$(GOPLATFORMS),pkg/gen/gljimports/gljimports_$(platform).go)

GO_VERSION := 1.19.3 # eventually, support multiple minor versions
GO_CMD := go$(GO_VERSION)

.PHONY: all
all: $(STDLIB_TARGETS) generate $(GLJIMPORTS)

.PHONY: gocmd
gocmd:
	@go$(GO_VERSION) version > /dev/null || (go install golang.org/dl/$(GO_VERSION) && $(GO_CMD) version)

.PHONY: generate
generate:
	@go generate ./...

pkg/gen/gljimports/gljimports_%.go: gocmd ./scripts/gen-gljimports.sh ./cmd/gen-import-interop/main.go \
					$(wildcard ./pkg/lang/*.go) $(wildcard ./pkg/runtime/*.go)
	@echo "Generating $@"
	@./scripts/gen-gljimports.sh $@ $* $(GO_CMD)

pkg/stdlib/glojure/%.glj: scripts/rewrite-core/originals/%.clj scripts/rewrite-core/run.sh scripts/rewrite-core/rewrite.clj
	@echo "Rewriting $< to $@"
	@mkdir -p $(dir $@)
	@scripts/rewrite-core/run.sh $< > $@

.PHONY: vet
vet:
	@go vet ./...

.PHONY: $(TEST_TARGETS)
$(TEST_TARGETS): gocmd
	@$(GO_CMD) run ./cmd/glj/main.go $(basename $@)

.PHONY: test
test: vet $(TEST_TARGETS)
