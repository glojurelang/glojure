
# Generate the core.glj file from Clojure's core.clj.
stdlib/glojure/core.glj: scripts/rewrite-core/rewrite.clj scripts/rewrite-core/run.sh
	@cd scripts/rewrite-core && ./run.sh > ../../stdlib/glojure/core.glj

vet:
	@go vet ./...
