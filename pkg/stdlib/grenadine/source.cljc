(ns grenadine.source
  "Recognize and load deps configuration from HTTP URLs."
  (:require [clojure.string :as str]))

(defn remote?
  "Return true when source is an HTTP or HTTPS URL."
  [source]
  (let [source (str/lower-case source)]
    (or (str/starts-with? source "http://")
        (str/starts-with? source "https://"))))

(defn- github-blob?
  [url]
  (boolean
   (re-find #"(?i)^https?://github[.]com/[^/]+/[^/]+/blob/.+" url)))

(defn- add-raw-query
  [url]
  (let [fragment-index (str/index-of url "#")
        base (if fragment-index (subs url 0 fragment-index) url)
        fragment (if fragment-index (subs url fragment-index) "")
        separator (if (str/includes? base "?") "&" "?")]
    (str base separator "raw=1" fragment)))

(defn request-url
  "Return the URL to fetch, requesting raw content for GitHub blob pages."
  [url]
  (if (github-blob? url)
    (add-raw-query url)
    url))

(defn fetch-text
  "Fetch an HTTP source and decode its successful response as UTF-8."
  [host source]
  (let [response ((:http-get host) (request-url source))
        status (:status response)]
    (cond
      (= 200 status)
      ((:bytes->utf8 host) (:body response))

      (or (nil? status) (zero? status))
      (throw (ex-info (str "cannot read " source ": request failed")
                      {:source source :status status}))

      :else
      (throw (ex-info (str "cannot read " source ": HTTP " status)
                      {:source source :status status})))))
