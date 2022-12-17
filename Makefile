
# Generate the core.glj file from Clojure's core.clj.
stdlib/glojure/core.glj: scripts/rewrite-core/rewrite.clj
	@cd scripts/rewrite-core && ./run.sh
