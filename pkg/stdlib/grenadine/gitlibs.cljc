;   Copyright (c) Rich Hickey. All rights reserved.
;   The use and distribution terms for this software are covered by the
;   Eclipse Public License 1.0 (http://opensource.org/licenses/eclipse-1.0.php)
;   which can be found in the file epl-v10.html at the root of this distribution.
;   By using this software in any fashion, you are agreeing to be bound by
;   the terms of this license.
;   You must not remove this notice, or any other, from this software.
;
;   Portable adaptation of org.clojure/tools.gitlibs at v2.6.217.
;   Grenadine adaptations Copyright 2026 Ingy döt Net, under EPL 1.0.
;   See Provenance.md for the exact source mapping and changes.

(ns grenadine.gitlibs
  "Portable tools.gitlibs-compatible Git cache operations."
  (:require [clojure.string :as str]))

(defn gitlibs-dir
  "Resolve the Git cache directory.

  Explicit :gitlibs-dir wins, followed by GRENADINE_GITLIBS_CACHE, GITLIBS,
  and ~/.gitlibs."
  [{:keys [host gitlibs-dir]}]
  (or gitlibs-dir
      (let [getenv (:getenv host)]
        (when getenv
          (let [value (getenv "GRENADINE_GITLIBS_CACHE")]
            (when (seq value) value))))
      (let [getenv (:getenv host)]
        (when getenv
          (let [value (getenv "GITLIBS")]
            (when (seq value) value))))
      (str ((:home-dir host)) "/.gitlibs")))

(defn- git-command [{:keys [host]}]
  (or (let [getenv (:getenv host)]
        (when getenv
          (let [value (getenv "GITLIBS_COMMAND")]
            (when (seq value) value))))
      "git"))

(defn clean-url
  "Convert a Git URL into the tools.gitlibs mirror-cache relative path."
  [url]
  (let [[scheme host path]
        (cond
          (str/starts-with? url "file://")
          ["file" nil (-> url (subs 7)
                          (str/replace #"^([^/])" "REL/$1"))]

          (str/includes? url "://")
          (let [[_ scheme host path]
                (re-matches
                 #"([a-z0-9+.-]+)://(?:(?:(?:[^@]+?)@)?([^/]+?)(?::[0-9]*)?)?(/[^:]+)"
                 url)]
            [scheme host path])

          (str/includes? url ":")
          (let [[_ host path]
                (re-matches #"(?:(?:[^@]+?)@)?(.+?):([^:]+)" url)]
            ["ssh" host path])

          :else
          ["file" nil (str/replace url #"^([^/])" "REL/$1")])
        clean-path (-> path
                       (str/replace #"[.]git/?$" "")
                       (str/replace "~" "_TILDE_"))
        parts (->> (concat [scheme host] (str/split clean-path #"/"))
                   (remove str/blank?)
                   (map #(get {"." "_DOT_" ".." "_DOTDOT_"} % %)))]
    (str/join "/" parts)))

(defn mirror-dir [url opts]
  (str (gitlibs-dir opts) "/_repos/" (clean-url url)))

(defn lib-dir [lib opts]
  (str (gitlibs-dir opts) "/libs/" (namespace lib) "/" (name lib)))

(defn checkout-dir [lib sha opts]
  (str (lib-dir lib opts) "/" sha))

(defn- run
  [opts args]
  (let [host (:host opts)
        run-process (:run-process host)]
    (when-not run-process
      (throw (ex-info "Git coordinates are not supported by this host"
                      {:type :grenadine.gitlibs/unsupported-host})))
    (let [result
          (run-process
           {:args (into [(git-command opts)] args)
            :env {"GIT_TERMINAL_PROMPT" "0"}})]
      result)))

(defn- git-run!
  [opts args message]
  (let [{:keys [exit err] :as result} (run opts args)]
    (when-not (zero? exit)
      (throw (ex-info (str message (when (seq (str/trim err))
                                     (str ": " (str/trim err))))
                      {:type :grenadine.gitlibs/git-failed
                       :args args
                       :exit exit
                       :err err})))
    result))

(defn ensure-mirror!
  [url opts]
  (let [host (:host opts)
        mirror (mirror-dir url opts)
        config (str mirror "/config")]
    (when-not ((:exists? host) config)
      ((:mkdirs! host) (subs mirror 0 (str/last-index-of mirror "/")))
      (git-run! opts ["clone" "--quiet" "--mirror" url mirror]
            (str "Unable to clone " url)))
    mirror))

(defn fetch!
  [url opts]
  (let [mirror (ensure-mirror! url opts)]
    (git-run! opts ["--git-dir" mirror "fetch" "--quiet" "--all" "--tags" "--prune"]
          (str "Unable to fetch " url))
    mirror))

(defn- parse-revision
  [mirror revision opts]
  (let [{:keys [exit out]}
        (run opts ["--git-dir" mirror "rev-parse" (str revision "^{commit}")])]
    (when (zero? exit) (str/trim out))))

(defn resolve-revision
  "Resolve a SHA, SHA prefix, or tag to a full commit SHA."
  [url revision opts]
  (let [mirror (ensure-mirror! url opts)]
    (or (parse-revision mirror revision opts)
        (do (fetch! url opts)
            (parse-revision mirror revision opts))
        (throw (ex-info (str "Unable to resolve Git revision " revision
                             " in " url)
                        {:type :grenadine.gitlibs/revision-not-found
                         :url url :revision revision})))))

(defn procure!
  "Ensure a tools.gitlibs-layout checkout and return its canonical path."
  [url lib sha opts]
  (let [host (:host opts)
        mirror (ensure-mirror! url opts)
        checkout (checkout-dir lib sha opts)]
    (when-not ((:exists? host) (str checkout "/.git"))
      ((:mkdirs! host) (lib-dir lib opts))
      (when ((:exists? host) checkout)
        ((or (:delete-tree! host) (:delete! host)) checkout))
      (git-run! opts ["--git-dir" mirror "worktree" "add" "--force" "--detach"
                  checkout sha]
            (str "Unable to check out " lib " at " sha))
      (git-run! opts ["-C" checkout "submodule" "update" "--init" "--recursive"
                  "--quiet"]
            (str "Unable to update submodules for " lib)))
    ((or (:canonical-path host) identity) checkout)))

(defn compare-revisions
  "Compare two full SHAs by ancestry. Descendants sort after ancestors."
  [url left right opts]
  (if (= left right)
    0
    (let [mirror (ensure-mirror! url opts)
          ancestor?
          (fn [a b]
            (= 0 (:exit (run opts ["--git-dir" mirror "merge-base"
                                   "--is-ancestor" a b]))))]
      (cond
        (ancestor? left right) -1
        (ancestor? right left) 1
        :else
        (throw (ex-info "Git revisions have no ancestry relationship"
                        {:type :grenadine.gitlibs/unrelated-revisions
                         :url url :left left :right right}))))))
