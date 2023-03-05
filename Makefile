
STDLIB := $(notdir $(wildcard scripts/rewrite-core/originals/*.clj))
STDLIB_ORIGINALS := $(addprefix scripts/rewrite-core/originals/,$(STDLIB))
STDLIB_TARGETS := $(addprefix stdlib/glojure/,$(STDLIB:.clj=.glj))

all: $(STDLIB_TARGETS)

stdlib/glojure/%.glj: scripts/rewrite-core/originals/%.clj scripts/rewrite-core/run.sh scripts/rewrite-core/rewrite.clj
	@echo "Rewriting $<"
	@scripts/rewrite-core/run.sh $< > $@

stdlib/glojure/%.glj: scripts/rewrite-core/originals/%.clj scripts/rewrite-core/run.sh scripts/rewrite-core/rewrite.clj
	@echo "Rewriting $<"
	@scripts/rewrite-core/run.sh $< > $@


# Generate the core.glj file from Clojure's core.clj.
# stdlib/glojure/core.glj: scripts/rewrite-core/rewrite.clj scripts/rewrite-core/run.sh
# 	@cd scripts/rewrite-core && ./run.sh "./originals/core.clj" > ../../stdlib/glojure/core.glj

# stdlib/glojure/core_print.glj: scripts/rewrite-core/rewrite.clj scripts/rewrite-core/run.sh
# 	@cd scripts/rewrite-core && ./run.sh "./core_print.clj"  > ../../stdlib/glojure/core_print.glj

# stdlib/glojure/test.glj: scripts/rewrite-core/rewrite.clj scripts/rewrite-core/run.sh
# 	@cd scripts/rewrite-core && ./run.sh "./test.clj"  > ../../stdlib/glojure/test.glj

vet:
	@go vet ./...
