;   Copyright (c) Rich Hickey. All rights reserved.
;   The use and distribution terms for this software are covered by the
;   Eclipse Public License 1.0 (http://opensource.org/licenses/eclipse-1.0.php)
;   which can be found in the file epl-v10.html at the root of this distribution.
;   By using this software in any fashion, you are agreeing to be bound by
;   the terms of this license.
;   You must not remove this notice, or any other, from this software.
;
;   Portable adaptation of clojure.tools.deps extension and coordinate logic
;   at v0.31.1642, plus canonicalization behavior from tools.deps.edn v0.9.48.
;   Grenadine adaptations Copyright 2026 Ingy döt Net, under EPL 1.0.
;   See Provenance.md for the exact source mapping and changes.

(ns grenadine.coordinate
  "Coordinate recognition and canonicalization shared by every resolver host."
  (:require [clojure.string :as str]
            [grenadine.gitlibs :as gitlibs]
            [grenadine.repo :as repo]
            [grenadine.version :as version]))

(def coordinate-keysets
  {:mvn #{:mvn/version}
   :git #{:git/url :git/sha :git/tag :sha :tag}
   :local #{:local/root}})

(defn coordinate-type
  [coordinate]
  (when-not (map? coordinate)
    (throw (ex-info (str "Coordinate must be a map: " (pr-str coordinate))
                    {:type :grenadine.coordinate/invalid-coordinate
                     :coordinate coordinate})))
  (let [keys (set (keys coordinate))
        matches
        (->> coordinate-keysets
             (keep (fn [[type identifying]]
                     (when (seq (filter keys identifying)) type)))
             vec)]
    (case (count matches)
      1 (first matches)
      0 (throw (ex-info (str "Coordinate has no recognized type: "
                             (pr-str coordinate))
                        {:type :grenadine.coordinate/unknown-coordinate
                         :coordinate coordinate}))
      (throw (ex-info (str "Coordinate type is ambiguous: "
                           (pr-str coordinate))
                      {:type :grenadine.coordinate/ambiguous-coordinate
                       :coordinate coordinate
                       :matches matches})))))

(defn split-lib
  [lib]
  (let [[group raw-artifact]
        (cond
          (symbol? lib) [(namespace lib) (name lib)]
          (string? lib)
          (if-let [slash (str/index-of lib "/")]
            [(subs lib 0 slash) (subs lib (inc slash))]
            [nil lib])
          :else
          (throw (ex-info (str "Library name must be a symbol or string: "
                               (pr-str lib))
                          {:type :grenadine.coordinate/invalid-lib :lib lib})))
        classifier-index (str/index-of raw-artifact "$")
        artifact (if classifier-index
                   (subs raw-artifact 0 classifier-index)
                   raw-artifact)
        classifier (when classifier-index
                     (subs raw-artifact (inc classifier-index)))]
    [(or group artifact) artifact classifier]))

(defn artifact-name
  [artifact classifier]
  (str artifact (when classifier (str "$" classifier))))

(defn lib-key
  [group artifact classifier]
  [group (artifact-name artifact classifier)])

(defn lib-symbol
  [group artifact classifier]
  (symbol group (artifact-name artifact classifier)))

(defn base-lib
  "Remove a Maven classifier while preserving the library's spelling."
  [lib]
  (if (symbol? lib)
    (let [raw-name (name lib)
          classifier-index (str/index-of raw-name "$")
          base-name (if classifier-index
                      (subs raw-name 0 classifier-index)
                      raw-name)]
      (if-let [lib-namespace (namespace lib)]
        (symbol lib-namespace base-name)
        (symbol base-name)))
    (if-let [classifier-index (str/index-of lib "$")]
      (subs lib 0 classifier-index)
      lib)))

(def git-url-hosts
  [["io.github." "https://github.com/" ".git"]
   ["com.github." "https://github.com/" ".git"]
   ["io.gitlab." "https://gitlab.com/" ".git"]
   ["com.gitlab." "https://gitlab.com/" ".git"]
   ["io.bitbucket." "https://bitbucket.org/" ".git"]
   ["org.bitbucket." "https://bitbucket.org/" ".git"]
   ["org.codeberg." "https://codeberg.org/" ".git"]
   ["page.codeberg." "https://codeberg.org/" ".git"]
   ["com.beanstalkapp." "https://" ".git"]
   ["io.beanstalkapp." "https://" ".git"]
   ["ht.sr." "https://git.sr.ht/" ""]])

(defn infer-git-url
  [lib]
  (when-let [lib-ns (namespace lib)]
    (some
     (fn [[prefix url-prefix suffix]]
       (when (str/starts-with? lib-ns prefix)
         (let [owner (subs lib-ns (count prefix))]
           (if (str/includes? prefix "beanstalkapp")
             (str url-prefix owner ".beanstalkapp.com/" (name lib) suffix)
             (str url-prefix owner "/" (name lib) suffix)))))
     git-url-hosts)))

(defn- full-sha? [value]
  (and (string? value)
       (= 40 (count value))
       (boolean (re-matches #"[0-9a-fA-F]{40}" value))))

(defn- join-path [base path]
  (if (or (nil? path) (= "" path))
    base
    (str (str/replace base #"/+$" "") "/" path)))

(defn canonical-local-root
  [root base-dir host]
  (let [candidate
        (if (or (str/starts-with? root "/")
                (boolean (re-matches #"[A-Za-z]:[\\/].*" root)))
          root
          (join-path (or base-dir ".") root))
        canonical ((or (:canonical-path host) (:absolute-path host) identity)
                   candidate)]
    (when-not ((:exists? host) canonical)
      (throw (ex-info (str "Local dependency does not exist: " canonical)
                      {:type :grenadine.coordinate/missing-local-root
                       :root canonical})))
    canonical))

(defn canonicalize
  "Return a canonical coordinate. Git canonicalization may fetch its mirror."
  [lib coordinate {:keys [host base-dir] :as opts}]
  (case (coordinate-type coordinate)
    :mvn
    (let [v (:mvn/version coordinate)]
      (when-not (and (string? v) (seq v))
        (throw (ex-info (str "Maven coordinate requires :mvn/version: " lib)
                        {:type :grenadine.coordinate/invalid-maven
                         :lib lib :coordinate coordinate})))
      (if (version/version-range? v)
        (assoc coordinate :mvn/version
               (or (version/exact-range-version v)
                   (repo/resolve-version-range
                    (let [[group artifact] (split-lib lib)]
                      {:group group :artifact artifact})
                    v opts)))
        coordinate))

    :git
    (let [coordinate (cond-> coordinate
                       (:sha coordinate) (assoc :git/sha (:sha coordinate))
                       (:tag coordinate) (assoc :git/tag (:tag coordinate)))
          coordinate (dissoc coordinate :sha :tag)
          url (or (:git/url coordinate) (infer-git-url lib))
          sha (:git/sha coordinate)
          tag (:git/tag coordinate)]
      (when-not url
        (throw (ex-info (str "Git coordinate has no URL and none can be inferred: " lib)
                        {:type :grenadine.coordinate/missing-git-url
                         :lib lib :coordinate coordinate})))
      (when-not (and (string? sha) (seq sha))
        (throw (ex-info (str "Git coordinate requires :git/sha: " lib)
                        {:type :grenadine.coordinate/missing-git-sha
                         :lib lib :coordinate coordinate})))
      (when (and (not tag) (not (full-sha? sha)))
        (throw (ex-info (str "Git SHA must be complete unless paired with :git/tag: " lib)
                        {:type :grenadine.coordinate/incomplete-git-sha
                         :lib lib :coordinate coordinate})))
      (let [revision (or tag sha)
            full (gitlibs/resolve-revision url revision opts)]
        (when-not (str/starts-with? (str/lower-case full)
                                    (str/lower-case sha))
          (throw (ex-info (str "Git SHA does not match resolved revision: " lib)
                          {:type :grenadine.coordinate/git-sha-mismatch
                           :lib lib :coordinate coordinate :resolved full})))
        (assoc coordinate :git/url url :git/sha full)))

    :local
    (let [root (:local/root coordinate)]
      (when-not (and (string? root) (seq root))
        (throw (ex-info (str "Local coordinate requires :local/root: " lib)
                        {:type :grenadine.coordinate/invalid-local
                         :lib lib :coordinate coordinate})))
      (assoc coordinate :local/root
             (canonical-local-root root base-dir host)))))

(defn dep-id
  [coordinate]
  (case (coordinate-type coordinate)
    :mvn [:mvn (:mvn/version coordinate)]
    :git [:git (:git/url coordinate) (:git/sha coordinate)]
    :local [:local (:local/root coordinate)]))

(defn compare-coordinates
  [lib left right opts]
  (let [left-type (coordinate-type left)
        right-type (coordinate-type right)]
    (when-not (= left-type right-type)
      (throw (ex-info (str "Coordinate types are incomparable for " lib)
                      {:type :grenadine.coordinate/incomparable
                       :lib lib :left left :right right})))
    (case left-type
      :mvn (version/compare-versions (:mvn/version left) (:mvn/version right))
      :git (do
             (when-not (= (:git/url left) (:git/url right))
               (throw (ex-info (str "Git URLs are incomparable for " lib)
                               {:type :grenadine.coordinate/incomparable
                                :lib lib :left left :right right})))
             (gitlibs/compare-revisions (:git/url left)
                                        (:git/sha left)
                                        (:git/sha right)
                                        opts))
      :local (if (= (:local/root left) (:local/root right))
               0
               (throw (ex-info (str "Local roots are incomparable for " lib)
                               {:type :grenadine.coordinate/incomparable
                                :lib lib :left left :right right}))))))
