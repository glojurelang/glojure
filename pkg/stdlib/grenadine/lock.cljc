(ns grenadine.lock
  "Pure Grenadine lockfile construction."
  (:require [clojure.string :as str]))

(def default-repos
  [{:id "central" :url "https://repo.maven.apache.org/maven2/"}
   {:id "clojars" :url "https://repo.clojars.org/"}])

(defn artifact-path
  [{:keys [group artifact version classifier]}]
  (let [directory (str (str/replace group "." "/")
                       "/" artifact "/" version "/")
        filename (str artifact "-" version
                      (when classifier (str "-" classifier))
                      ".jar")]
    (str directory filename)))

(defn- repo-url
  [repo]
  (if (map? repo) (:url repo) repo))

(defn- gav
  [{:keys [group artifact version]}]
  [group artifact version])

(defn emit-lock
  "Convert a resolution into stable lockfile data.

  `:pom-fn` identifies graph-only `pom` packaging. `:repo-fn` may choose a
  repository index for each coordinate and defaults to the first repository.
  `:integrity` is an optional GAV-keyed map containing :sha256 and :size."
  [resolution {:keys [repos pom-fn repo-fn integrity]
               :or {repos default-repos
                    repo-fn (fn [_] 0)}}]
  (let [repo-urls (mapv repo-url repos)
        artifacts
        (->> (:selected resolution)
             vals
             (map :coords)
             (sort-by (juxt :group :artifact :version))
             (keep
              (fn [coords]
                (let [pom (when pom-fn (pom-fn coords))
                      packaging (or (:packaging pom) "jar")]
                  (when (not= packaging "pom")
                    (merge
                     {:group (:group coords)
                      :artifact (:artifact coords)
                      :version (:version coords)
                      :packaging packaging
                      :path (artifact-path coords)
                      :repo (repo-fn coords)}
                     (select-keys (get integrity (gav coords))
                                  [:sha256 :size]))))))
             vec)]
    {:lock/version 1
     :repos repo-urls
     :artifacts artifacts}))

(defn lock->classpath
  "Return local artifact paths without touching the filesystem."
  [lock {:keys [local-repo]}]
  (let [local-repo
        (loop [value local-repo]
          (if (and (seq value)
                   (= \/ (nth value (dec (count value)))))
            (recur (subs value 0 (dec (count value))))
            value))]
    (mapv #(str local-repo
              "/" (:path %))
          (:artifacts lock))))
