
STDLIB := $(notdir $(wildcard scripts/rewrite-core/originals/*.clj))
STDLIB_ORIGINALS := $(addprefix scripts/rewrite-core/originals/,$(STDLIB))
STDLIB_TARGETS := $(addprefix stdlib/glojure/,$(STDLIB:.clj=.glj))

all: $(STDLIB_TARGETS) generate

.PHONY:generate
generate:
	@go generate ./...

stdlib/glojure/%.glj: scripts/rewrite-core/originals/%.clj scripts/rewrite-core/run.sh scripts/rewrite-core/rewrite.clj
	@echo "Rewriting $<"
	@scripts/rewrite-core/run.sh $< > $@

stdlib/glojure/%.glj: scripts/rewrite-core/originals/%.clj scripts/rewrite-core/run.sh scripts/rewrite-core/rewrite.clj
	@echo "Rewriting $<"
	@scripts/rewrite-core/run.sh $< > $@

vet:
	@go vet ./...
