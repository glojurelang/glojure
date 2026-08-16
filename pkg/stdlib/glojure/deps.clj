(ns glojure.deps
  "Glojure dependency facade backed by Grenadine.

  Installed JARs are safely extracted and their source roots are appended with
  `clojure.core/add-load-path`."
  (:require [clojure.string :as str]
            [grenadine.require-deps :as required]
            [grenadine.runtime :as runtime]
            [glojure.deps.host :as host]))

(defonce ^:private basis (atom {:libs {} :classpath {} :classpath-roots []
                                :grenadine/loaded {}}))

(defonce ^:private required-state
  (atom {:coordinates {} :namespaces {}}))

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
               (assoc :repos (:mvn/repos deps-map))

               (:gitlibs/dir deps-map)
               (assoc :gitlibs-dir (:gitlibs/dir deps-map))))))

(defn sync-deps
  ([] (sync-deps "deps.edn" nil))
  ([path] (sync-deps path nil))
  ([path opts]
   (let [index (max (or (str/last-index-of path "/") -1)
                    (or (str/last-index-of path "\\") -1))]
     (add-deps (read-string (slurp path))
               (assoc (or opts {}) :base-dir
                      (if (neg? index) "." (subs path 0 index)))))))

(defn- required-host
  []
  (let [runtime-host (host/host)]
    {:home-dir (:home-dir runtime-host)
     :file-exists? (:regular-file? runtime-host)
     :mkdirs! (:mkdirs! runtime-host)
     :delete! (:delete! runtime-host)
     :atomic-move! (:atomic-move! runtime-host)
     :read-text
     (fn [path]
       ((:bytes->utf8 runtime-host) ((:read-bytes runtime-host) path)))
     :download!
     (fn [url path]
       (let [{:keys [status body]} ((:http-get runtime-host) url)]
         (when (and (<= 200 status 299) body)
           ((:write-bytes! runtime-host) path body)
           true)))}))

(defn- read-first-form
  [source]
  (binding [*read-eval* false]
    (read-string source)))

(defn- loaded-namespace
  [identity]
  (get-in @required-state [:coordinates identity]))

(defn- load-required!
  [coordinate namespace-symbol load!]
  (let [identity (:identity coordinate)
        loaded-coordinate (get-in @required-state
                                  [:namespaces namespace-symbol])]
    (cond
      (= identity (:identity loaded-coordinate)) namespace-symbol
      loaded-coordinate
      (required/namespace-conflict! namespace-symbol loaded-coordinate coordinate)
      :else
      (do
        (load!)
        (swap! required-state
               (fn [state]
                 (-> state
                     (assoc-in [:coordinates identity] namespace-symbol)
                     (assoc-in [:namespaces namespace-symbol] coordinate))))
        namespace-symbol))))

(defn prepare-required!
  "Internal hook used by clojurestar.deps/require-deps."
  [coordinate options]
  (or
   (loaded-namespace (:identity coordinate))
   (case (:provider coordinate)
     :mvn
     (load-required!
      coordinate (:namespace coordinate)
      #(do
         (add-libs {(:lib coordinate) {:mvn/version (:version coordinate)}}
                   (when-let [repository (:mvn/local-repo options)]
                     {:local-repo repository}))
         (require (:namespace coordinate))))

     :gist
     (let [{:keys [path source]}
           (required/acquire-gist! (required-host) options coordinate)
           namespace-symbol
           (required/gist-namespace coordinate (read-first-form source))]
       (load-required! coordinate namespace-symbol
                       #(let [caller (ns-name *ns*)]
                          (try
                            (load-file path)
                            (finally (in-ns caller)))))))))
