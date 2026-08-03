(ns grenadine.core
  "Portable Maven dependency resolution."
  (:require [grenadine.expander :as expander]
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
  (let [pom-fn (or (:pom-fn opts) (cached-pom-provider opts))
        resolution (graph/resolve-graph deps (assoc opts :pom-fn pom-fn))
        initial-lock (lock/emit-lock resolution (assoc opts :pom-fn pom-fn))
        fetched (repo/fetch-lock! initial-lock opts)]
    (when (seq (:failed fetched))
      (throw (ex-info "Failed to install Maven artifacts"
                      {:type :grenadine.core/install-failed
                       :failed (:failed fetched)})))
    (let [final-lock (:lock fetched)
          extraction
          (when (:source-roots? opts)
            (repo/prepare-source-roots! final-lock opts))]
      (when (seq (:failed extraction))
        (throw (ex-info "Failed to prepare Maven source roots"
                        {:type :grenadine.core/extraction-failed
                         :failed (:failed extraction)})))
      {:classpath (lock/lock->classpath
                   final-lock
                   {:local-repo (repo/local-repo opts)})
       :fetched (:fetched fetched)
       :cached (:cached fetched)
       :source-roots (:roots extraction)
       :lock final-lock
       :resolution resolution
       :warnings (vec (concat (:warnings resolution)
                              (:warnings fetched)))})))
