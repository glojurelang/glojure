
STDLIB := $(notdir $(wildcard scripts/rewrite-core/originals/*.clj))
STDLIB_ORIGINALS := $(addprefix scripts/rewrite-core/originals/,$(STDLIB))
STDLIB_TARGETS := $(addprefix pkg/stdlib/glojure/,$(STDLIB:.clj=.glj))

all: $(STDLIB_TARGETS) generate pkg/gen/gljimports/gljimports.go

.PHONY:generate
generate:
	@go generate ./...

pkg/gen/gljimports/gljimports.go: ./scripts/gen-gljimports.sh ./cmd/gen-import-interop/main.go $(wildcard ./pkg/*/*.go)
	@echo "Generating $@"
	@./scripts/gen-gljimports.sh $@

pkg/stdlib/glojure/%.glj: scripts/rewrite-core/originals/%.clj scripts/rewrite-core/run.sh scripts/rewrite-core/rewrite.clj
	@echo "Rewriting $<"
	@scripts/rewrite-core/run.sh $< > $@

pkg/stdlib/glojure/%.glj: scripts/rewrite-core/originals/%.clj scripts/rewrite-core/run.sh scripts/rewrite-core/rewrite.clj
	@echo "Rewriting $<"
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
