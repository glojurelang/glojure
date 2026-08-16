(ns clojurestar.deps
  "Dialect-neutral dynamic dependency loading."
  (:require [grenadine.require-deps :as required]
   #?(:gobb [gobb.deps :as implementation]
      :glj [glojure.deps :as implementation]
      :jolt [jolt.deps :as implementation])))

(defn add-deps
  "Add the dependencies in a deps.edn map to the running dialect.

  This portable facade deliberately returns nil. Use the dialect-specific
  dependency namespace when backend-specific options or results are needed."
  [deps-map]
  (implementation/add-deps deps-map)
  nil)

(defn require-deps
  "Acquire and load one or more quoted dependency libspecs.

  An optional leading map accepts :mvn/local-repo and :cache-dir. Libspecs
  support :as and an explicit :refer vector. Dependencies are prepared and
  required from left to right."
  [& arguments]
  (let [[options libspecs]
        (if (map? (first arguments))
          [(required/parse-options (first arguments)) (rest arguments)]
          [{} arguments])]
    (when (empty? libspecs)
      (throw
       (ex-info "require-deps requires at least one quoted libspec vector"
                {:type :clojurestar.deps/invalid-require})))
    (doseq [libspec libspecs]
      (let [{:keys [coordinate] alias-symbol :as refer-symbols :refer}
            (required/parse-libspec libspec)
            namespace-symbol
            (implementation/prepare-required! coordinate options)]
        (when alias-symbol
          (alias alias-symbol namespace-symbol))
        (when refer-symbols
          (refer namespace-symbol :only refer-symbols))))
    nil))
