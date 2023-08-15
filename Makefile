
STDLIB_ORIGINALS_DIR := scripts/rewrite-core/originals
STDLIB_ORIGINALS := $(shell find $(STDLIB_ORIGINALS_DIR) -name '*.clj')
# STDLIB_ORIGINALS := $(wildcard scripts/rewrite-core/originals/**/*.clj)
# STDLIB_ORIGINALS += $(wildcard scripts/rewrite-core/originals/*.clj)
STDLIB := $(STDLIB_ORIGINALS:scripts/rewrite-core/originals/%=%)
STDLIB_ORIGINALS := $(addprefix scripts/rewrite-core/originals/,$(STDLIB))
STDLIB_TARGETS := $(addprefix pkg/stdlib/glojure/,$(STDLIB:.clj=.glj))

GOPLATFORMS := darwin_arm64 darwin_amd64 linux_arm64 linux_amd64 windows
GLJIMPORTS=$(foreach platform,$(GOPLATFORMS),pkg/gen/gljimports/gljimports_$(platform).go)

all: $(STDLIB_TARGETS) generate $(GLJIMPORTS)

.PHONY:generate
generate:
	@go generate ./...

pkg/gen/gljimports/gljimports_%.go: ./scripts/gen-gljimports.sh ./cmd/gen-import-interop/main.go $(wildcard ./pkg/*/*.go)
	@echo "Generating $@"
	@./scripts/gen-gljimports.sh $@ $*

pkg/stdlib/glojure/%.glj: scripts/rewrite-core/originals/%.clj scripts/rewrite-core/run.sh scripts/rewrite-core/rewrite.clj
	@echo "Rewriting $< to $@"
	@mkdir -p $(dir $@)
	@scripts/rewrite-core/run.sh $< > $@

.PHONY: vet
vet:
	@go vet ./...

.PHONY: test
test: vet
	@go test ./...
	@go run ./cmd/glj/main.go ./test/glojure/test_glojure/basic.glj
	@go run ./cmd/glj/main.go ./test/glojure/test_glojure/import.glj
	@go run ./cmd/glj/main.go ./test/glojure/test_glojure/printer.glj
	@go run ./cmd/glj/main.go ./test/glojure/test_glojure/builtins.glj
	@go run ./cmd/glj/main.go ./test/glojure/test_glojure/core/async/basic.glj
