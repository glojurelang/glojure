
STDLIB := $(notdir $(wildcard scripts/rewrite-core/originals/*.clj))
STDLIB_ORIGINALS := $(addprefix scripts/rewrite-core/originals/,$(STDLIB))
STDLIB_TARGETS := $(addprefix stdlib/glojure/,$(STDLIB:.clj=.glj))

all: $(STDLIB_TARGETS) generate gen/gljimports/gljimports.go

.PHONY:generate
generate:
	@go generate ./...

gen/gljimports/gljimports.go: ./cmd/gen-import-interop/main.go $(wildcard ./value/*.go)
	@go run ./cmd/gen-import-interop/main.go > $@

stdlib/glojure/%.glj: scripts/rewrite-core/originals/%.clj scripts/rewrite-core/run.sh scripts/rewrite-core/rewrite.clj
	@echo "Rewriting $<"
	@scripts/rewrite-core/run.sh $< > $@

stdlib/glojure/%.glj: scripts/rewrite-core/originals/%.clj scripts/rewrite-core/run.sh scripts/rewrite-core/rewrite.clj
	@echo "Rewriting $<"
	@scripts/rewrite-core/run.sh $< > $@

vet:
	@go vet ./...

.PHONY: test
test:
	# @go run ./cmd/glj/main.go ./test/glojure/test_glojure/basic.glj
	# @go run ./cmd/glj/main.go ./test/glojure/test_glojure/import.glj
	@go run ./cmd/glj/main.go ./test/glojure/test_glojure/printer.glj

