(ns glojure.deps
  "Glojure dependency facade backed by Grenadine.

  Installed JARs are safely extracted and their source roots are appended with
  `clojure.core/add-load-path`."
  (:require [grenadine.runtime :as runtime]
            [glojure.deps.host :as host]))

(defonce ^:private basis (atom {}))

(defn current-basis [] (runtime/current-basis basis))

(defn- add-roots!
  [roots]
  (let [add-load-path (resolve 'clojure.core/add-load-path)]
    (when-not add-load-path
      (throw (ex-info "Glojure does not expose clojure.core/add-load-path"
                      {:type :grenadine.runtime/missing-load-path-hook})))
    (doseq [root roots] (add-load-path root))))

(defn add-libs
  ([libs] (add-libs libs nil))
  ([libs opts]
   (let [opts (or opts {})]
     (runtime/add-libs! basis add-roots! libs
                        (assoc opts :host (or (:host opts) (host/host)))))))

(defn add-lib
  ([lib coordinate] (add-lib lib coordinate nil))
  ([lib coordinate opts] (add-libs {lib coordinate} opts)))

(defn add-deps
  ([deps-map] (add-deps deps-map nil))
  ([deps-map opts]
   (add-libs (or (:deps deps-map) {})
             (cond-> (or opts {})
               (:mvn/local-repo deps-map)
               (assoc :local-repo (:mvn/local-repo deps-map))

               (:mvn/repos deps-map)
               (assoc :repos (:mvn/repos deps-map))))))

(defn sync-deps
  ([] (sync-deps "deps.edn" nil))
  ([path] (sync-deps path nil))
  ([path opts]
   (add-deps (read-string (slurp path)) opts)))
