;   Copyright (c) Rich Hickey. All rights reserved.
;   The use and distribution terms for this software are covered by the
;   Eclipse Public License 1.0 (http://opensource.org/licenses/eclipse-1.0.php)
;   which can be found in the file epl-v10.html at the root of this distribution.
;   By using this software in any fashion, you are agreeing to be bound by
;   the terms of this license.
;   You must not remove this notice, or any other, from this software.
;
;   Portable adaptation of clojure.tools.deps basis and manifest-extension
;   organization at v0.31.1642.
;   Grenadine adaptations Copyright 2026 Ingy döt Net, under EPL 1.0.
;   See Provenance.md for the exact source mapping and changes.

(ns grenadine.basis
  "Portable mixed-coordinate resolution and tools.deps-shaped bases."
  (:require [clojure.string :as str]
            [grenadine.coordinate :as coordinate]
            [grenadine.expander :as expander]
            [grenadine.gitlibs :as gitlibs]
            [grenadine.graph :as graph]
            [grenadine.lock :as lock]
            [grenadine.pom :as pom]
            [grenadine.repo :as repo]))

(defn- join-path [base path]
  (if (or (nil? path) (= "" path))
    base
    (str (str/replace base #"/+$" "") "/" path)))

(defn- parent-path [path]
  (if-let [index (str/last-index-of path "/")]
    (subs path 0 index)
    "."))

(defn- read-text [host path]
  ((:bytes->utf8 host) ((:read-bytes host) path)))

(defn- read-edn [host path]
  ((or (:read-edn host) read-string) (read-text host path)))

(defn- canonical [host path]
  ((or (:canonical-path host) (:absolute-path host) identity) path))

(defn- relative-to [root path]
  (let [root (str/replace root #"/+$" "")
        prefix (str root "/")]
    (cond
      (= root path) ""
      (str/starts-with? path prefix) (subs path (count prefix))
      :else path)))

(defn- safe-deps-root
  [host root relative]
  (let [root (canonical host root)
        nested (canonical host (join-path root relative))
        prefix (str (str/replace root #"/+$" "") "/")]
    (when-not (or (= root nested) (str/starts-with? nested prefix))
      (throw (ex-info (str ":deps/root escapes dependency root: " relative)
                      {:type :grenadine.basis/unsafe-deps-root
                       :root root :deps/root relative})))
    nested))

(defn- pom-lib
  [group artifact classifier]
  (coordinate/lib-symbol group artifact classifier))

(defn- pom-deps
  [model]
  (into []
        (keep
         (fn [{:keys [group artifact classifier version scope optional exclusions]}]
           (when (and group artifact version
                      (not (#{"test" "provided" "system"} scope))
                      (not optional))
             [(pom-lib group artifact classifier)
              (cond-> {:mvn/version version}
                (seq exclusions)
                (assoc :exclusions
                       (set (map #(symbol (:group %) (:artifact %))
                                 exclusions))))])))
        (:deps model)))

(defn- raw-pom-deps
  [text]
  (let [raw (pom/parse-pom text)
        properties (:properties raw)
        interpolate #(pom/interpolate-string % properties)]
    (into []
          (keep
           (fn [{:keys [group artifact classifier version scope optional exclusions]}]
             (let [group (interpolate group)
                   artifact (interpolate artifact)
                   classifier (interpolate classifier)
                   version (interpolate version)]
               (when (and group artifact version
                          (not (#{"test" "provided" "system"} scope))
                          (not optional)
                          (not (str/includes? version "${")))
                 [(pom-lib group artifact classifier)
                  (cond-> {:mvn/version version}
                    (seq exclusions)
                    (assoc :exclusions
                           (set (map #(symbol (:group %) (:artifact %))
                                     exclusions))))]))))
          (:dependencies raw))))

(defn- normalize-deps
  [deps base-dir opts]
  (when (and (some? deps) (not (map? deps)))
    (throw (ex-info ":deps must be a map"
                    {:type :grenadine.basis/invalid-deps :deps deps})))
  (into []
        (map (fn [[lib coord]]
               [lib (when coord
                      (coordinate/canonicalize
                       lib coord (assoc opts :base-dir base-dir)))]))
        (or deps {})))

(defn- detect-directory-manifest
  [lib coord root opts]
  (let [host (:host opts)
        requested (:deps/manifest coord)
        deps-path (join-path root "deps.edn")
        pom-path (join-path root "pom.xml")
        manifest
        (cond
          (= requested :deps) :deps
          (= requested :pom) :pom
          requested
          (throw (ex-info (str "Unsupported :deps/manifest for " lib ": " requested)
                          {:type :grenadine.basis/unsupported-manifest
                           :lib lib :manifest requested}))
          ((:exists? host) deps-path) :deps
          ((:exists? host) pom-path) :pom
          :else nil)]
    (when-not manifest
      (throw (ex-info (str "Dependency has neither deps.edn nor pom.xml: " root)
                      {:type :grenadine.basis/missing-manifest
                       :lib lib :root root})))
    (case manifest
      :deps
      (do
        (when-not ((:exists? host) deps-path)
          (throw (ex-info (str "Missing deps.edn: " deps-path)
                          {:type :grenadine.basis/missing-manifest-file
                           :lib lib :path deps-path})))
        (let [data (read-edn host deps-path)]
          (when (and (contains? data :deps) (not (map? (:deps data))))
            (throw (ex-info (str ":deps must be a map in " deps-path)
                            {:type :grenadine.basis/invalid-deps
                             :lib lib :path deps-path})))
          {:manifest :deps
           :root root
           :data data
           :children (normalize-deps (:deps data) root opts)
           :paths (mapv #(canonical host (join-path root %))
                        (or (:paths data) ["src"]))}))

      :pom
      (do
        (when-not ((:exists? host) pom-path)
          (throw (ex-info (str "Missing pom.xml: " pom-path)
                          {:type :grenadine.basis/missing-manifest-file
                           :lib lib :path pom-path})))
        {:manifest :pom
         :root root
         :children (normalize-deps
                    (into {} (raw-pom-deps (read-text host pom-path)))
                    root opts)
         :paths (mapv #(canonical host (join-path root %))
                      ["src/main/java" "src/main/clojure"
                       "src/main/resources"])}))))

(defn- local-jar-info
  [lib coord opts]
  (let [host (:host opts)
        jar (:local/root coord)
        bytes ((:read-bytes host) jar)
        digest ((:digest host) :sha256 bytes)
        destination (str jar ".grenadine/" digest)]
    ((:extract-jar! host) jar destination)
    (let [find-files (:find-files host)
          poms (when find-files
                 (find-files destination
                             #(and (str/includes? % "/META-INF/")
                                   (str/ends-with? % "/pom.xml"))))
          pom-path (first poms)]
      {:manifest :jar
       :root jar
       :source-root destination
       :children (if pom-path
                   (normalize-deps
                    (into {} (raw-pom-deps (read-text host pom-path)))
                    (parent-path jar) opts)
                   [])
       :paths [jar]})))

(defn- coordinate-engine
  [opts]
  (let [host (:host opts)
        cache (atom {})
        pom-fn
        (or (:pom-fn opts)
            (let [fetch-pom (or (:fetch-pom opts) (repo/pom-fetcher opts))
                  poms (atom {})]
              (fn [coords]
                (if-let [entry (find @poms coords)]
                  (val entry)
                  (let [value (pom/effective-pom coords fetch-pom)]
                    (swap! poms assoc coords value)
                    value)))))
        info
        (fn info [lib coord]
          (let [key [lib (coordinate/dep-id coord)]]
            (if-let [entry (find @cache key)]
              (val entry)
              (let [value
                    (case (coordinate/coordinate-type coord)
                      :mvn
                      (let [[group artifact classifier]
                            (coordinate/split-lib lib)
                            coords (cond->
                                   {:group group :artifact artifact
                                    :version (:mvn/version coord)}
                                     classifier
                                     (assoc :classifier classifier))]
                        {:manifest :mvn
                         :children (normalize-deps
                                    (into {} (pom-deps (pom-fn coords)))
                                    (:base-dir opts) opts)})

                      :git
                      (let [checkout
                            (gitlibs/checkout-dir lib (:git/sha coord) opts)
                            present? ((:exists? host) (str checkout "/.git"))
                            checkout (gitlibs/procure! (:git/url coord) lib
                                                       (:git/sha coord) opts)
                            _ (when (and (not present?)
                                         (:on-install-coordinate opts))
                                ((:on-install-coordinate opts)
                                 {:lib lib :coordinate coord}))
                            root (if-let [relative (:deps/root coord)]
                                   (safe-deps-root host checkout relative)
                                   checkout)]
                        (assoc (detect-directory-manifest lib coord root opts)
                               :checkout checkout
                               :cached? present?))

                      :local
                      (let [root (:local/root coord)]
                        (if ((:directory? host) root)
                          (let [effective
                                (if-let [relative (:deps/root coord)]
                                  (safe-deps-root host root relative)
                                  root)]
                            (assoc (detect-directory-manifest lib coord effective opts)
                                   :cached? true))
                          (assoc (local-jar-info lib coord opts) :cached? true))))]
                (swap! cache assoc key value)
                value))))]
    {:info info :pom-fn pom-fn}))

(defn- selected-paths
  [trace lib coord]
  (let [id (coordinate/dep-id coord)
        paths (get-in trace [:vmap lib :paths id])]
    (or paths #{})))

(defn- parents-data
  [trace libs]
  (into {}
        (map
         (fn [[lib coord]]
           (let [paths (selected-paths trace lib coord)]
             [lib {:parents (set (map vec paths))
                   :dependents (vec (distinct (keep peek paths)))}])))
        libs))

(defn- classpath-lib-order
  [lib-map]
  (->> lib-map
       (mapcat
        (fn [[lib {:keys [parents]}]]
          (map #(conj (vec %) lib) parents)))
       sort
       (sort-by count)
       (map peek)
       distinct
       vec))

(defn- maven-resolution
  [libs]
  {:selected
   (into {}
         (keep
          (fn [[lib coord]]
            (when (= :mvn (coordinate/coordinate-type coord))
              (let [[group artifact classifier]
                    (coordinate/split-lib lib)]
                [(coordinate/lib-key group artifact classifier)
                 {:coords (cond->
                           {:group group :artifact artifact
                            :version (:mvn/version coord)}
                            classifier (assoc :classifier classifier))}]))))
         libs)})

(defn calc-basis
  "Calculate a tools.deps-shaped basis from an already merged deps map."
  [deps-map {:keys [host resolve-args classpath-args base-dir] :as opts}]
  (when-not host
    (throw (ex-info "calc-basis requires :host"
                    {:type :grenadine.basis/missing-host})))
  (let [base-dir (or base-dir ".")
        opts (assoc opts :base-dir base-dir)
        {:keys [info pom-fn]} (coordinate-engine opts)
        deps (merge (:deps deps-map) (:extra-deps resolve-args))
        top (normalize-deps deps base-dir opts)
        canonical-map
        (fn [m] (into {} (normalize-deps m base-dir opts)))
        expansion
        (expander/expand-deps
         top
         {:coord-id (fn [_ coord] (coordinate/dep-id coord))
          :coord-deps (fn [lib coord] (:children (info lib coord)))
          :base-lib coordinate/base-lib
          :compare-versions
          (fn [lib left right]
            (coordinate/compare-coordinates lib left right opts))
          :known-coordinate? (fn [coord] (some? coord))
          :override-deps (canonical-map (:override-deps resolve-args))
          :default-deps (canonical-map (:default-deps resolve-args))
          :trace? true
          :fail-on-incomparable? true})
        libs
        (if (and (not= :tools-deps (:mediation opts :tools-deps))
                 (every? #(= :mvn (coordinate/coordinate-type (second %))) top))
          (let [legacy
                (graph/resolve-graph
                 (into {} top)
                 {:pom-fn pom-fn :mediation (:mediation opts)})]
            (into {}
                  (map
                   (fn [[_ occurrence]]
                     (let [{:keys [group artifact classifier version]}
                           (:coords occurrence)]
                       [(coordinate/lib-symbol group artifact classifier)
                        {:mvn/version version}])))
                  (:selected legacy)))
          (:libs expansion))
        resolution (maven-resolution libs)
        initial-lock (lock/emit-lock resolution
                                     (assoc opts :pom-fn pom-fn))
        fetched (if (false? (:fetch-artifacts? opts))
                  {:lock initial-lock :fetched [] :cached []
                   :failed [] :warnings []}
                  (repo/fetch-lock! initial-lock opts))
        _ (when (seq (:failed fetched))
            (throw (ex-info "Failed to install Maven artifacts"
                            {:type :grenadine.basis/install-failed
                             :failed (:failed fetched)})))
        final-lock (:lock fetched)
        artifacts
        (into {}
              (map (fn [{:keys [group artifact classifier] :as entry}]
                     [(coordinate/lib-symbol group artifact classifier) entry]))
              (:artifacts final-lock))
        parent-data (parents-data (:trace expansion) libs)
        expansion-libs (filter #(contains? libs %) (:order expansion))
        lib-map
        (into {}
              (map
               (fn [lib]
                 (let [coord (get libs lib)
                       type (coordinate/coordinate-type coord)
                       details (info lib coord)
                       artifact (get artifacts lib)
                       paths
                       (case type
                         :mvn (if artifact
                                [(str (canonical host (repo/local-repo opts))
                                      "/" (:path artifact))]
                                [])
                         :git (:paths details)
                         :local (:paths details))]
                   [lib (merge (cond-> coord
                                 (not= type :mvn)
                                 (assoc :deps/root (:root details)))
                               {:deps/manifest (:manifest details)
                                :paths (vec paths)}
                               (cond-> (get parent-data lib)
                                 (empty? (get-in parent-data [lib :dependents]))
                                 (dissoc :dependents)))]))
               expansion-libs))
        ordered-libs (classpath-lib-order lib-map)
        extra-paths (:extra-paths classpath-args)
        project-paths (or (:replace-paths classpath-args)
                          (:paths deps-map)
                          [])
        project-roots
        (mapv #(canonical host (join-path base-dir %))
              (concat extra-paths project-paths))
        overrides (:classpath-overrides classpath-args)
        dependency-roots
        (mapcat
         (fn [lib]
           (if-let [override (get overrides lib)]
             [(canonical host (join-path base-dir override))]
             (get-in lib-map [lib :paths])))
         ordered-libs)
        classpath-roots (vec (distinct (concat project-roots dependency-roots)))
        path-key-map
        (into {}
              (map-indexed
               (fn [index root]
                 [root {:path-key (if (< index (count extra-paths))
                                    :extra-paths :paths)}])
               project-roots))
        lib-path-map
        (into {}
              (mapcat
               (fn [[lib data]]
                 (map (fn [path] [path {:lib-name lib}]) (:paths data)))
               lib-map))
        source-extraction
        (when (:source-roots? opts)
          (repo/prepare-source-roots! final-lock opts))
        non-maven-source-roots
        (when (:source-roots? opts)
          (mapcat
           (fn [lib]
             (let [coord (get libs lib)
                   type (coordinate/coordinate-type coord)
                   details (info lib coord)]
               (case type
                 :mvn []
                 :git (:paths details)
                 :local (if (= :jar (:manifest details))
                          [(:source-root details)]
                          (:paths details)))))
           ordered-libs))
        lock-v2
        (assoc final-lock
               :lock/version 2
               :libs
               (mapv
                (fn [lib]
                  (let [coord (get libs lib)
                        type (coordinate/coordinate-type coord)
                        details (info lib coord)
                        artifact (get artifacts lib)
                        classpath
                        (case type
                          :mvn (if artifact
                                 [{:type :mvn :path (:path artifact)}]
                                 [])
                          :git (mapv (fn [path]
                                       {:type :git
                                        :path (relative-to (:checkout details)
                                                           path)})
                                     (:paths details))
                          :local (mapv (fn [path] {:type :local :path path})
                                       (:paths details)))]
                    {:lib lib
                     :coord coord
                     :deps/manifest (:manifest details)
                     :classpath classpath}))
                ordered-libs))
        fetched-libs
        (set (map #(coordinate/lib-symbol (:group %) (:artifact %)
                                          (:classifier %))
                  (:fetched fetched)))
        cached-libs
        (set (map #(coordinate/lib-symbol (:group %) (:artifact %)
                                          (:classifier %))
                  (:cached fetched)))
        installed-libs
        (vec
         (filter
          (fn [lib]
            (let [coord (get libs lib)]
              (case (coordinate/coordinate-type coord)
                :mvn (contains? fetched-libs lib)
                :git (not (:cached? (info lib coord)))
                :local false)))
          ordered-libs))
        already-libs
        (vec
         (filter
          (fn [lib]
            (let [coord (get libs lib)]
              (case (coordinate/coordinate-type coord)
                :mvn (contains? cached-libs lib)
                :git (:cached? (info lib coord))
                :local true)))
          ordered-libs))]
    (merge deps-map
           {:libs lib-map
            :classpath (merge path-key-map lib-path-map)
            :classpath-roots classpath-roots
            :grenadine/lock lock-v2
            :grenadine/fetched (:fetched fetched)
            :grenadine/cached (:cached fetched)
            :grenadine/installed-libs installed-libs
            :grenadine/already-libs already-libs
            :grenadine/warnings
            (vec (concat (:warnings expansion) (:warnings fetched)))
            :grenadine/resolution expansion
            :grenadine/source-roots
            (vec (concat (:roots source-extraction)
                         non-maven-source-roots))})))
