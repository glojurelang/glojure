
all: stdlib/glojure/core.glj stdlib/glojure/core_print.glj

# Generate the core.glj file from Clojure's core.clj.
stdlib/glojure/core.glj: scripts/rewrite-core/rewrite.clj scripts/rewrite-core/run.sh
	@cd scripts/rewrite-core && ./run.sh "./core.clj" > ../../stdlib/glojure/core.glj

stdlib/glojure/core_print.glj: scripts/rewrite-core/rewrite.clj scripts/rewrite-core/run.sh
	@cd scripts/rewrite-core && ./run.sh "./core_print.clj"  > ../../stdlib/glojure/core_print.glj

vet:
	@go vet ./...
