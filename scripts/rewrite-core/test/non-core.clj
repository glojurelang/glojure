(ns rewrite-test.non-core
  (:refer-clojure :exclude [read]))

(defn read [path]
  (os.ReadFile path))

(defn typed-value ^long [^long x] x)

(defn adjacent [x]
  (Math/nextUp x))
