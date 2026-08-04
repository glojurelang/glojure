(ns grenadine.core
  "Portable Maven, Git, and local dependency resolution."
  (:require [grenadine.basis :as basis]
            [grenadine.expander :as expander]
            [grenadine.graph :as graph]
            [grenadine.lock :as lock]
            [grenadine.pom :as pom]
            [grenadine.repo :as repo]))

(def parse-pom pom/parse-pom)
(def interpolate pom/interpolate-string)
(def expand-deps expander/expand-deps)
(def emit-lock lock/emit-lock)
(def lock->classpath lock/lock->classpath)
(def fetch-lock! repo/fetch-lock!)
(def prepare-source-roots! repo/prepare-source-roots!)
(def calc-basis basis/calc-basis)

(defn- cached-pom-provider
  [opts]
  (let [fetch-pom (or (:fetch-pom opts) (repo/pom-fetcher opts))
        cache (atom {})]
    (fn [coords]
      (let [key [(:group coords) (:artifact coords) (:version coords)]]
        (if-let [cached (get @cache key)]
          cached
          (let [effective (pom/effective-pom coords fetch-pom)]
            (swap! cache assoc key effective)
            effective))))))

(defn effective-pom
  "Build an effective POM using `:fetch-pom`, a coordinate-to-POM function."
  [coords opts]
  ((cached-pom-provider opts) coords))

(defn resolve-graph
  "Resolve deps using an explicit `:pom-fn` or a repository-backed provider."
  [deps opts]
  (graph/resolve-graph
   deps
   (if (:pom-fn opts)
     opts
     (assoc opts :pom-fn (cached-pom-provider opts)))))

(defn install!
  "Resolve and install deps, returning classpath, lock, and warnings.

  Pass `:source-roots? true` to also extract and return portable Clojure source
  roots for non-JVM runtimes."
  [deps opts]
  (let [result (basis/calc-basis {:deps deps} opts)]
    {:classpath (:classpath-roots result)
     :fetched (:grenadine/fetched result)
     :cached (:grenadine/cached result)
     :installed-libs (:grenadine/installed-libs result)
     :already-libs (:grenadine/already-libs result)
     :source-roots (:grenadine/source-roots result)
     :lock (:grenadine/lock result)
     :resolution (:grenadine/resolution result)
     :basis result
     :warnings (:grenadine/warnings result)}))
