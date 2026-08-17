(ns grenadine.require-deps
  "Portable parsing and Gist acquisition for clojurestar.deps/require-deps."
  (:require [clojure.string :as str]))

(defn- fail!
  [message data]
  (throw (ex-info message (assoc data :type :clojurestar.deps/invalid-require))))

(defn- nonblank-string?
  [value]
  (and (string? value) (not (str/blank? value))))

(defn- safe-part?
  [value]
  (and (nonblank-string? value)
       (boolean (re-matches #"[A-Za-z0-9_.-]+" value))
       (not (#{"." ".."} value))))

(defn- safe-owner?
  [value]
  (and (nonblank-string? value)
       (boolean (re-matches #"[A-Za-z0-9-]+" value))))

(defn- safe-gist-id?
  [value]
  (and (nonblank-string? value)
       (boolean (re-matches #"[A-Fa-f0-9]+" value))))

(defn- safe-filename?
  [value]
  (and (safe-part? value)
       (boolean (re-matches #"[A-Za-z0-9_.-]+\.cljc?" value))))

(defn- unqualified-symbol?
  [value]
  (and (symbol? value)
       (nil? (namespace value))
       (not (str/blank? (name value)))))

(defn- namespace-symbol?
  [value]
  (and (unqualified-symbol? value)
       (boolean (re-matches #"[A-Za-z0-9_][A-Za-z0-9_.-]*" (name value)))))

(defn- parse-revision
  [coordinate body]
  (let [at (str/last-index-of body "@")]
    (if (nil? at)
      [body nil]
      (let [base (subs body 0 at)
            revision (subs body (inc at))]
        (when (str/includes? base "@")
          (fail! (str "Malformed Gist coordinate: " coordinate)
                 {:coordinate coordinate}))
        (when-not (boolean (re-matches #"[A-Fa-f0-9]{40}" revision))
          (fail! "Pinned Gist revisions must be full 40-character commit SHAs"
                 {:coordinate coordinate :revision revision}))
        [base revision]))))

(defn- parse-gist
  [coordinate body]
  (let [[body revision] (parse-revision coordinate body)
        parts (str/split body #"/" -1)]
    (when-not (<= 2 (count parts) 3)
      (fail! (str "Malformed Gist coordinate: " coordinate)
             {:coordinate coordinate}))
    (let [[owner id filename] parts]
      (when-not (and (safe-owner? owner)
                     (safe-gist-id? id)
                     (or (nil? filename) (safe-filename? filename)))
        (fail! (str "Malformed or unsafe Gist coordinate: " coordinate)
               {:coordinate coordinate}))
      {:provider :gist
       :owner owner
       :id id
       :filename filename
       :revision revision
       :coordinate coordinate
       :identity [:gist owner id filename revision]})))

(defn- parse-maven
  [coordinate]
  (let [[_ group artifact version namespace-name]
        (re-matches #"mvn:([^/@]+)/([^/@]+)@([^/]+)/([^/]+)" coordinate)
        namespace-symbol (when namespace-name (symbol namespace-name))]
    (when-not (and (safe-part? group)
                   (safe-part? artifact)
                   (nonblank-string? version)
                   (not (str/includes? version "@"))
                   (namespace-symbol? namespace-symbol))
      (fail! (str "Malformed Maven require coordinate: " coordinate)
             {:coordinate coordinate}))
    {:provider :mvn
     :lib (symbol group artifact)
     :version version
     :namespace namespace-symbol
     :coordinate coordinate
     :identity [:mvn group artifact version namespace-name]}))

(defn parse-coordinate
  "Parse a supported require coordinate into portable data."
  [coordinate]
  (when-not (nonblank-string? coordinate)
    (fail! "A require coordinate must be a non-empty string"
           {:coordinate coordinate}))
  (cond
    (str/starts-with? coordinate "mvn:")
    (parse-maven coordinate)

    (str/starts-with? coordinate "gist:")
    (parse-gist coordinate (subs coordinate 5))

    :else
    (if-let [[_ owner id]
             (re-matches #"https://gist\.github\.com/([^/]+)/([^/]+)/?" coordinate)]
      (parse-gist coordinate (str owner "/" id))
      (fail! (str "Unsupported require coordinate: " coordinate)
             {:coordinate coordinate}))))

(defn parse-libspec
  "Validate a require-deps libspec vector."
  [libspec]
  (when-not (and (vector? libspec) (seq libspec))
    (fail! "A require-deps libspec must be a non-empty vector"
           {:libspec libspec}))
  (let [coordinate (parse-coordinate (first libspec))
        option-forms (rest libspec)]
    (when (odd? (count option-forms))
      (fail! "Libspec options must be keyword/value pairs"
             {:libspec libspec}))
    (loop [forms option-forms result {:coordinate coordinate} seen #{}]
      (if (empty? forms)
        result
        (let [[option value & more] forms]
          (when-not (keyword? option)
            (fail! "Libspec option names must be keywords"
                   {:libspec libspec :option option}))
          (when (contains? seen option)
            (fail! (str "Duplicate libspec option: " option)
                   {:libspec libspec :option option}))
          (case option
            :as
            (if (unqualified-symbol? value)
              (recur more (assoc result :as value) (conj seen option))
              (fail! ":as requires an unqualified alias symbol"
                     {:libspec libspec :alias value}))

            :refer
            (cond
              (= :all value)
              (fail! ":refer :all is not supported"
                     {:libspec libspec :refer value})

              (and (vector? value)
                   (seq value)
                   (every? unqualified-symbol? value)
                   (= (count value) (count (distinct value))))
              (recur more (assoc result :refer value) (conj seen option))

              :else
              (fail! ":refer requires a vector of unqualified symbols"
                     {:libspec libspec :refer value}))

            (fail! (str "Unsupported libspec option: " option)
                   {:libspec libspec :option option})))))))

(defn parse-options
  "Validate the optional leading require-deps options map."
  [options]
  (let [options (or options {})
        supported #{:mvn/local-repo :cache-dir}]
    (when-not (map? options)
      (fail! "require-deps options must be a map" {:options options}))
    (when-let [unknown (first (remove supported (keys options)))]
      (fail! (str "Unsupported require-deps option: " unknown)
             {:options options :option unknown}))
    (doseq [option supported
            :let [value (get options option)]
            :when (contains? options option)]
      (when-not (nonblank-string? value)
        (fail! (str option " must be a non-empty string")
               {:options options :option option :value value})))
    options))

(defn gist-raw-url
  "Return the raw.githubusercontent.com URL for a parsed Gist coordinate."
  [{:keys [owner id filename revision] :as coordinate}]
  (when-not (= :gist (:provider coordinate))
    (fail! "Expected a parsed Gist coordinate" {:coordinate coordinate}))
  (str "https://gist.githubusercontent.com/" owner "/" id "/raw"
       (when revision (str "/" revision))
       (when filename (str "/" filename))))

(defn cache-root
  "Resolve the clojurestar cache root from options and a host home directory."
  [host options]
  (or (:cache-dir options)
      (when-let [home ((:home-dir host))]
        (str home "/.cache/clojurestar"))
      (fail! "Unable to determine the default clojurestar cache directory"
             {:option :cache-dir})))

(defn gist-cache-path
  "Return the persistent cache file for a parsed Gist coordinate."
  [host options {:keys [owner id filename revision] :as coordinate}]
  (when-not (= :gist (:provider coordinate))
    (fail! "Expected a parsed Gist coordinate" {:coordinate coordinate}))
  (str (cache-root host options) "/gist/" owner "/" id "/"
       (or revision "latest") "/" (or filename "source.clj")))

(defn- parent-path
  [path]
  (subs path 0 (str/last-index-of path "/")))

(defn acquire-gist!
  "Acquire a Gist through a runtime host and return its cache path and source.

  HOST supplies :home-dir, :file-exists?, :mkdirs!, :download!, :read-text,
  :atomic-move!, and :delete!. Pinned files reuse persistent cache; latest
  files are fetched by the first preparation in every process."
  [host options coordinate]
  (let [target (gist-cache-path host options coordinate)
        pinned? (some? (:revision coordinate))
        cached? ((:file-exists? host) target)]
    (if (and pinned? cached?)
      {:path target :source ((:read-text host) target) :cached? true}
      (let [temporary (str target ".grenadine.part")]
        ((:mkdirs! host) (parent-path target))
        ((:delete! host) temporary)
        (try
          (when-not ((:download! host) (gist-raw-url coordinate) temporary)
            (throw
             (ex-info (str "Unable to download Gist source: "
                           (:coordinate coordinate))
                      {:type :clojurestar.deps/gist-http-failure
                       :coordinate (:coordinate coordinate)
                       :url (gist-raw-url coordinate)})))
          ((:atomic-move! host) temporary target)
          {:path target :source ((:read-text host) target) :cached? false}
          (finally
            ((:delete! host) temporary)))))))

(defn gist-namespace
  "Extract and validate the namespace from a Gist's first source form."
  [coordinate first-form]
  (let [namespace-symbol
        (when (and (seq? first-form)
                   (= 'ns (first first-form))
                   (symbol? (second first-form)))
          (second first-form))]
    (when-not namespace-symbol
      (throw
       (ex-info "A selected Gist file must begin with an ns form"
                {:type :clojurestar.deps/gist-missing-ns
                 :coordinate (:coordinate coordinate)})))
    namespace-symbol))

(defn namespace-conflict!
  [namespace-symbol loaded-coordinate requested-coordinate]
  (throw
   (ex-info (str "Namespace " namespace-symbol
                 " is already loaded from another coordinate")
            {:type :clojurestar.deps/namespace-conflict
             :namespace namespace-symbol
             :loaded-coordinate loaded-coordinate
             :requested-coordinate requested-coordinate})))
