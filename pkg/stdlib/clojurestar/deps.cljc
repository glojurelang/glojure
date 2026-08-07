(ns clojurestar.deps
  "Dialect-neutral dynamic dependency loading."
  (:require
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
