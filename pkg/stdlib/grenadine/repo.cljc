(ns grenadine.repo
  "Effectful Maven repository operations expressed through a host function map."
  (:require [clojure.string :as str]
            [grenadine.lock :as lock]
            [grenadine.version :as version]
            [grenadine.xml :as xml]))

(defn- require-host
  [host keys]
  (doseq [key keys]
    (when-not (fn? (get host key))
      (throw (ex-info (str "Host is missing " key)
                      {:type :grenadine.repo/incomplete-host
                       :key key}))))
  host)

(defn local-repo
  "Return the Maven local repository path.

  An explicit `:local-repo` wins, followed by
  `GRENADINE_LOCAL_REPOSITORY`, then `$HOME/.m2/repository`."
  [{:keys [host local-repo]}]
  (or local-repo
      (let [getenv (:getenv host)
            configured (when (fn? getenv)
                         (getenv "GRENADINE_LOCAL_REPOSITORY"))]
        (when (seq configured) configured))
      (str ((:home-dir host)) "/.m2/repository")))

(defn- trim-trailing-slashes
  [value]
  (loop [value value]
    (if (and (seq value) (= \/ (nth value (dec (count value)))))
      (recur (subs value 0 (dec (count value))))
      value)))

(defn- parent-path
  [path]
  (if-let [slash (str/last-index-of path "/")]
    (subs path 0 slash)
    "."))

(defn- pom-path
  [{:keys [group artifact version]}]
  (str (str/replace group "." "/")
       "/" artifact "/" (version/snapshot-base-version version) "/"
       artifact "-" version ".pom"))

(defn- remote-url
  [repo path]
  (str (trim-trailing-slashes
        (if (map? repo) (:url repo) repo))
       "/" path))

(defn- child-elements
  [node tag]
  (filter #(and (map? %) (= tag (:tag %))) (:content node)))

(defn- child-element
  [node tag]
  (first (child-elements node tag)))

(defn- element-text
  [node]
  (when node
    (let [value (str/trim (apply str (filter string? (:content node))))]
      (when (seq value) value))))

(defn- metadata-data
  [source]
  (let [metadata (xml/parse source)
        versioning (child-element metadata :versioning)
        release (element-text (child-element versioning :release))
        latest (element-text (child-element versioning :latest))
        versions-node (child-element versioning :versions)
        versions
        (->> (child-elements versions-node :version)
             (keep element-text)
             vec)]
    {:release release :latest latest :versions versions}))

(defn- metadata-version
  [source]
  (let [{:keys [release latest versions]} (metadata-data source)]
    (or release
        latest
        (reduce
         (fn [selected candidate]
           (if (or (nil? selected) (version/newer? candidate selected))
             candidate
             selected))
         nil
         (remove #(str/ends-with? (str/upper-case %) "-SNAPSHOT")
                 versions)))))

(defn resolve-version-range
  "Resolve a Maven version range to the highest matching repository version."
  [{:keys [group artifact] :as coords} range-spec {:keys [host repos]}]
  (require-host host [:http-get :bytes->utf8])
  (let [path (str (str/replace group "." "/")
                  "/" artifact "/maven-metadata.xml")
        repos (or repos lock/default-repos)
        parsed-range (version/parse-version-range range-spec)
        candidates
        (mapcat
         (fn [repository]
           (let [url (remote-url repository path)
                 response ((:http-get host) url)]
             (if (= 200 (:status response))
               (try
                 (:versions
                  (metadata-data ((:bytes->utf8 host) (:body response))))
                 (catch Exception error
                   (throw
                    (ex-info (str "Invalid Maven metadata for "
                                  group "/" artifact)
                             {:type :grenadine.repo/invalid-metadata
                              :coords coords
                              :url url}
                             error))))
               [])))
         repos)
        matching (filter #(version/in-range? % parsed-range)
                         (distinct candidates))
        selected
        (reduce
         (fn [current candidate]
           (if (or (nil? current) (version/newer? candidate current))
             candidate
             current))
         nil
         matching)]
    (or selected
        (throw
         (ex-info (str "No Maven version satisfies " group "/" artifact
                       " " range-spec)
                  {:type :grenadine.repo/version-range-not-found
                   :coords coords
                   :range range-spec})))))

(defn latest-version
  "Return the latest Maven release for a group/artifact coordinate.

  Repositories are tried in order. Metadata `release` wins, followed by
  `latest`, then the highest listed non-SNAPSHOT version."
  [{:keys [group artifact] :as coords} {:keys [host repos]}]
  (require-host host [:http-get :bytes->utf8])
  (let [path (str (str/replace group "." "/")
                  "/" artifact "/maven-metadata.xml")
        repos (or repos lock/default-repos)]
    (loop [remaining repos]
      (if-let [repository (first remaining)]
        (let [url (remote-url repository path)
              response ((:http-get host) url)]
          (if (= 200 (:status response))
            (let [candidate
                  (try
                    (metadata-version ((:bytes->utf8 host) (:body response)))
                    (catch Exception error
                      (throw
                       (ex-info (str "Invalid Maven metadata for "
                                     group "/" artifact)
                                {:type :grenadine.repo/invalid-metadata
                                 :coords coords
                                 :url url}
                                error))))]
              (if candidate
                candidate
                (recur (next remaining))))
            (recur (next remaining))))
        (throw
         (ex-info (str "No Maven release found for " group "/" artifact)
                  {:type :grenadine.repo/version-not-found
                   :coords coords}))))))

(defn- successful?
  [response]
  (= 200 (:status response)))

(defn- write-atomically!
  [host target bytes]
  (let [temporary (str target ".grenadine.part")]
    ((:mkdirs! host) (parent-path target))
    (when ((:exists? host) temporary)
      ((:delete! host) temporary))
    ((:write-bytes! host) temporary bytes)
    ((:atomic-move! host) temporary target)))

(defn pom-fetcher
  "Return a coordinate-to-POM-text function backed by local Maven cache and
  ordered HTTP repositories."
  [{:keys [host repos] :as opts}]
  (require-host host
                [:http-get :read-bytes :write-bytes! :bytes->utf8
                 :exists? :mkdirs! :atomic-move! :delete! :home-dir])
  (let [base (trim-trailing-slashes (local-repo opts))
        repos (or repos lock/default-repos)]
    (fn [coords]
      (let [relative (pom-path coords)
            target (str base "/" relative)]
        (if ((:exists? host) target)
          ((:bytes->utf8 host) ((:read-bytes host) target))
          (loop [remaining repos]
            (when-let [repo (first remaining)]
              (let [response ((:http-get host) (remote-url repo relative))]
                (if (successful? response)
                  (do
                    (write-atomically! host target (:body response))
                    ((:bytes->utf8 host) (:body response)))
                  (recur (next remaining)))))))))))

(defn- first-word
  [value]
  (let [value (str/trim value)
        end (first
             (keep-indexed
              (fn [i c] (when (or (= c \space) (= c \tab)) i))
              value))]
    (if end (subs value 0 end) value)))

(defn- sha1-sidecar
  [host url]
  (let [response ((:http-get host) (str url ".sha1"))]
    (when (successful? response)
      (let [value (first-word
                   ((:bytes->utf8 host) (:body response)))]
        (when (seq value) (str/lower-case value))))))

(defn- integrity
  [host bytes]
  {:sha256 (str/lower-case ((:digest host) :sha256 bytes))
   :size ((:byte-count host) bytes)})

(defn- verify-bytes
  [host artifact bytes remote-sha1]
  (let [actual (integrity host bytes)
        expected-sha256 (some-> (:sha256 artifact) str/lower-case)
        actual-sha1
        (when (and (nil? expected-sha256) remote-sha1)
          (str/lower-case ((:digest host) :sha1 bytes)))
        valid?
        (cond
          expected-sha256 (= expected-sha256 (:sha256 actual))
          remote-sha1 (= remote-sha1 actual-sha1)
          :else true)]
    {:valid? valid?
     :actual actual
     :verified-by
     (cond expected-sha256 :sha256 remote-sha1 :sha1 :else nil)}))

(defn- artifact-url
  [lock artifact]
  (when-let [repo (get (:repos lock) (:repo artifact))]
    (remote-url repo (:path artifact))))

(defn- fetch-artifact
  [host lock artifact]
  (let [preferred (:repo artifact)
        indices (concat [preferred]
                        (remove #(= preferred %)
                                (range (count (:repos lock)))))]
    (loop [remaining indices last-response nil]
      (if-let [repo-index (first remaining)]
        (if-let [repo (get (:repos lock) repo-index)]
          (let [url (remote-url repo (:path artifact))
                response ((:http-get host) url)]
            (if (successful? response)
              {:repo repo-index :url url :response response}
              (recur (next remaining) response)))
          (recur (next remaining) last-response))
        {:response last-response}))))

(defn fetch-lock!
  "Install every artifact in a lock.

  Existing SHA-256 values are mandatory when supplied. Otherwise a remote
  SHA-1 sidecar is used when available. The returned `:lock` is enriched with
  computed SHA-256 and size values even though those fields remain optional in
  accepted input locks."
  [lock {:keys [host on-install] :as opts}]
  (require-host host
                [:http-get :read-bytes :write-bytes! :bytes->utf8
                 :digest :byte-count :exists? :mkdirs! :atomic-move!
                 :delete! :home-dir])
  (let [base (trim-trailing-slashes (local-repo opts))]
    (loop [remaining (:artifacts lock)
           enriched []
           fetched []
           cached []
           failed []
           warnings []]
      (if-let [artifact (first remaining)]
        (let [target (str base "/" (:path artifact))
              present? ((:exists? host) target)
              download (when-not present?
                         (fetch-artifact host lock artifact))
              url (or (:url download) (artifact-url lock artifact))
              response (:response download)
              artifact (if-let [repo-index (:repo download)]
                         (assoc artifact :repo repo-index)
                         artifact)
              bytes (if present?
                      ((:read-bytes host) target)
                      (when (successful? response) (:body response)))]
          (if (nil? bytes)
            (recur (next remaining) enriched fetched cached
                   (conj failed
                         {:artifact artifact
                          :reason (if url :download-failed :missing-repository)
                          :status (:status response)})
                   warnings)
            (let [remote-sha1
                  (when (and (nil? (:sha256 artifact))
                             (not present?)
                             url)
                    (sha1-sidecar host url))
                  verification
                  (verify-bytes host artifact bytes remote-sha1)]
              (if-not (:valid? verification)
                (recur (next remaining) enriched fetched cached
                       (conj failed
                             {:artifact artifact
                              :reason :checksum-mismatch
                              :actual (:actual verification)})
                       warnings)
                (do
                  (when-not present?
                    (write-atomically! host target bytes)
                    (when (fn? on-install)
                      (on-install artifact)))
                  (recur
                   (next remaining)
                   (conj enriched (merge artifact (:actual verification)))
                   (if present? fetched (conj fetched artifact))
                   (if present? (conj cached artifact) cached)
                   failed
                   (if (or present? (:verified-by verification))
                     warnings
                     (conj warnings
                           {:artifact artifact
                            :warning :unverified-artifact}))))))))
        {:lock (assoc lock :artifacts enriched)
         :fetched fetched
         :cached cached
         :failed failed
         :warnings warnings}))))

(defn prepare-source-roots!
  "Extract installed JARs into digest-keyed source directories.

  The host owns safe, atomic ZIP extraction. Returns roots in lock order.
  When `:source-libs` is supplied, only those library symbols are extracted."
  [lock {:keys [host source-libs] :as opts}]
  (require-host host
                [:read-bytes :digest :exists? :extract-jar! :home-dir])
  (let [base (trim-trailing-slashes (local-repo opts))
        artifacts
        (if source-libs
          (filter
           (fn [{:keys [group artifact classifier]}]
             (contains? source-libs
                        (symbol group
                                (str artifact
                                     (when classifier
                                       (str "$" classifier))))))
           (:artifacts lock))
          (:artifacts lock))]
    (loop [remaining artifacts roots [] failed []]
      (if-let [artifact (first remaining)]
        (let [jar (str base "/" (:path artifact))]
          (if-not ((:exists? host) jar)
            (recur (next remaining)
                   roots
                   (conj failed {:artifact artifact :reason :missing-artifact}))
            (let [bytes ((:read-bytes host) jar)
                  sha256 (or (:sha256 artifact)
                             ((:digest host) :sha256 bytes))
                  destination (str jar ".grenadine/" sha256)]
              ((:extract-jar! host) jar destination)
              (recur (next remaining)
                     (conj roots destination)
                     failed))))
        {:roots roots :failed failed}))))
