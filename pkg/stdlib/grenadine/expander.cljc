;   Copyright (c) Rich Hickey. All rights reserved.
;   The use and distribution terms for this software are covered by the
;   Eclipse Public License 1.0 (http://opensource.org/licenses/eclipse-1.0.php)
;   which can be found in the file epl-v10.html at the root of this distribution.
;   By using this software in any fashion, you are agreeing to be bound by
;   the terms of this license.
;   You must not remove this notice, or any other, from this software.
;
;   Portable adaptation of clojure.tools.deps at v0.31.1642.
;   Grenadine adaptations Copyright 2026 Ingy döt Net and contributors,
;   under EPL 1.0.
;   See Provenance.md for the exact source mapping and changes.

(ns grenadine.expander
  "Portable tools.deps-style dependency tree expansion.

  Coordinate behavior is supplied with functions, so Maven, Git, local, and
  test coordinates can share the same traversal and selection algorithm.")

(defn- intersection
  [left right]
  (set (filter right left)))

(defn- difference
  [left right]
  (set (remove right left)))

(defn- excluded?
  [base-lib exclusions path lib]
  (let [lib (base-lib lib)]
    (loop [search path]
      (when (seq search)
        (if (get-in exclusions [search lib])
          true
          (recur (pop search)))))))

(defn- update-excl
  "Update exclusions and cuts for a new coordinate, another occurrence of an
  existing coordinate, or an omitted coordinate."
  [lib coord coord-id use-path include? reason exclusions cuts]
  (let [coordinate-exclusions
        (when (contains? coord :exclusions)
          (set (:exclusions coord)))]
    (cond
      include?
      (if (nil? coordinate-exclusions)
        {:exclusions exclusions
         :cuts cuts
         :child-predicate (constantly true)}
        {:exclusions (assoc exclusions use-path coordinate-exclusions)
         :cuts (assoc cuts [lib coord-id] coordinate-exclusions)
         :child-predicate
         (fn [child]
           (not (contains? coordinate-exclusions child)))})

      (= reason :same-version)
      (let [prior-cut (set (get cuts [lib coord-id]))
            new-cut (intersection (set coordinate-exclusions) prior-cut)]
        {:exclusions
         (if (seq coordinate-exclusions)
           (assoc exclusions use-path coordinate-exclusions)
           exclusions)
         :cuts (assoc cuts [lib coord-id] new-cut)
         :child-predicate (difference prior-cut new-cut)})

      :else
      {:exclusions exclusions :cuts cuts})))

(defn- add-version
  [version-map lib coord path coord-id]
  (-> (or version-map {})
      (assoc-in [lib :versions coord-id] coord)
      (update-in
       [lib :paths]
       (fn [paths]
         (merge-with into {coord-id #{path}} paths)))))

(defn- select-version
  [version-map lib coord-id direct?]
  (update-in version-map [lib] merge
             (cond-> {:select coord-id}
               direct? (assoc :top true))))

(defn- selected-version
  [version-map lib]
  (get-in version-map [lib :select]))

(defn- selected-coord
  [version-map lib]
  (get-in version-map [lib :versions (selected-version version-map lib)]))

(defn- selected-paths
  [version-map lib]
  (get-in version-map [lib :paths (selected-version version-map lib)]))

(defn- parent-missing?
  [version-map parent-path]
  (loop [path parent-path alternatives nil]
    (if (seq path)
      (let [lib (last path)
            parent (vec (butlast path))
            {:keys [paths select]} (get version-map lib)
            selected-paths (get paths select)]
        (if (contains? selected-paths parent)
          (recur parent
                 (concat alternatives
                         (remove #(= parent %) selected-paths)))
          (if (seq alternatives)
            (recur (first alternatives) (rest alternatives))
            true)))
      false)))

(defn- path-prefix?
  [prefix path]
  (= prefix (vec (take (count prefix) path))))

(defn- deselect-orphans
  [version-map omitted-paths]
  (reduce
   (fn [result [lib {:keys [select paths]}]]
     (let [paths-to-selected (get paths select)]
       (if (every?
            (fn [path]
              (some #(path-prefix? % path) omitted-paths))
            paths-to-selected)
         (update-in result [lib] dissoc :select)
         result)))
   version-map
   version-map))

(defn- warn!
  [warnings on-warning warning]
  (swap! warnings conj warning)
  (when on-warning (on-warning warning)))

(defn- not-comparable!
  [lib candidate selected warnings on-warning error]
  (warn! warnings on-warning
         {:warning :versions-not-comparable
          :lib lib
          :selected selected
          :candidate candidate
          ;; the message alone: warnings are data that hosts render with
          ;; pr-str, so an exception's printed form would drag in its class
          ;; name and ex-data — and on Glojure it carries no message at all
          :message (ex-message error)})
  false)

(defn- newer?
  [lib candidate selected compare-versions warnings on-warning
   fail-on-incomparable?]
  ;; Glojure resolves neither Exception nor Throwable; `go/any` is the catch-all
  ;; it takes, the same split grenadine.test-support/throws? already makes.
  #?(:glj
     (try
       (pos? (compare-versions lib candidate selected))
       (catch go/any error
         (if fail-on-incomparable?
           (throw error)
           (not-comparable! lib candidate selected warnings on-warning error))))

     :default
     (try
       (pos? (compare-versions lib candidate selected))
       (catch Exception error
         (if fail-on-incomparable?
           (throw error)
           (not-comparable! lib candidate selected warnings on-warning error))))))

(defn- include-coord?
  "Decide whether to select a coordinate and return the reason and updated
  version map. This is the portable counterpart of tools.deps/include-coord?."
  [version-map lib coord coord-id path exclusions
   base-lib compare-versions warnings on-warning fail-on-incomparable?]
  (cond
    (empty? path)
    {:include? true
     :reason :new-top-dep
     :version-map
     (-> version-map
         (add-version lib coord path coord-id)
         (select-version lib coord-id true))}

    (excluded? base-lib exclusions path lib)
    {:include? false :reason :excluded :version-map version-map}

    (get-in version-map [lib :top])
    {:include? false :reason :use-top :version-map version-map}

    (parent-missing? version-map path)
    {:include? false :reason :parent-omitted :version-map version-map}

    (nil? (selected-version version-map lib))
    {:include? true
     :reason :new-dep
     :version-map
     (-> version-map
         (add-version lib coord path coord-id)
         (select-version lib coord-id false))}

    (= coord-id (selected-version version-map lib))
    {:include? false
     :reason :same-version
     :version-map (add-version version-map lib coord path coord-id)}

    (newer? lib coord (selected-coord version-map lib)
            compare-versions warnings on-warning fail-on-incomparable?)
    {:include? true
     :reason :newer-version
     :version-map
     (-> version-map
         (add-version lib coord path coord-id)
         (deselect-orphans
          (set (map #(conj % lib) (selected-paths version-map lib))))
         (select-version lib coord-id false))}

    :else
    {:include? false :reason :older-version :version-map version-map}))

(defn- next-path
  [pending queue index]
  (if-let [path (first pending)]
    {:path path
     :pending (next pending)
     :queue queue
     :index index}
    (if (< index (count queue))
      (let [item (nth queue index)
            index (inc index)]
        (if (:grenadine.expander/children item)
          (let [{:keys [children parent-path child-predicate]} item
                paths
                (->> children
                     (filter (fn [[lib _]] (child-predicate lib)))
                     (map #(conj parent-path %)))]
            (recur paths queue index))
          {:path item
           :pending nil
           :queue queue
           :index index}))
      {:path nil :pending nil :queue queue :index index})))

(defn- selected-libraries
  [version-map]
  (reduce
   (fn [result [lib {:keys [select versions]}]]
     (if select
       (assoc result lib (get versions select))
       result))
   {}
   version-map))

(defn expand-deps
  "Expand dependencies with tools.deps-compatible version selection.

  Required options are `:coord-id`, `:coord-deps`, and `:compare-versions`.
  Optional coordinate maps `:override-deps` and `:default-deps` are applied at
  every occurrence. `:provided-libs` names host-supplied libraries that are
  satisfied without selecting their coordinates or expanding their children.
  The result contains selected `:libs`, stable inclusion `:order`, collected
  `:warnings`, and, with `:trace?`, a tools.deps-shaped `:trace` containing
  `:log` and `:vmap`."
  [deps {:keys [coord-id coord-deps compare-versions known-coordinate?
                base-lib override-deps default-deps trace? on-warning
                fail-on-incomparable? provided-libs]
         :or {known-coordinate? some?
              base-lib identity
              override-deps {}
              default-deps {}
              provided-libs #{}}}]
  (when-not (and (ifn? coord-id)
                 (ifn? coord-deps)
                 (ifn? compare-versions))
    (throw
     (ex-info "expand-deps requires coordinate functions"
              {:type :grenadine.expander/missing-coordinate-function})))
  (let [warnings (atom [])
        dependency-cache (atom {})]
    (loop [pending nil
           queue (mapv vector deps)
           index 0
           version-map nil
           exclusions nil
           cuts nil
           order []
           trace []]
      (let [{:keys [path pending queue index]}
            (next-path pending queue index)]
        (if path
          (let [[lib original-coordinate] (peek path)
                parents (pop path)
                use-path (conj parents lib)
                coordinate
                (or (get override-deps lib)
                    original-coordinate
                    (get default-deps lib))]
            (if (contains? provided-libs (base-lib lib))
              (recur pending queue index version-map exclusions cuts
                     order
                     (if trace?
                       (conj trace
                             {:path (vec parents)
                              :lib lib
                              :coord coordinate
                              :orig-coord original-coordinate
                              :include false
                              :reason :provided})
                       trace))
              (if (or (nil? coordinate)
                    (not (known-coordinate? coordinate)))
                (do
                  (warn! warnings on-warning
                         {:warning :unsupported-coordinate
                          :lib lib
                          :coordinate coordinate})
                  (recur pending queue index version-map exclusions cuts
                         order trace))
                (let [id (coord-id lib coordinate)
                      decision
                      (include-coord?
                       version-map lib coordinate id parents exclusions
                       base-lib compare-versions warnings on-warning
                       fail-on-incomparable?)
                      include? (:include? decision)
                      version-map (:version-map decision)
                      reason (:reason decision)
                      update
                      (update-excl
                       lib coordinate id use-path include? reason
                       exclusions cuts)
                      child-predicate (:child-predicate update)
                      children
                      (when child-predicate
                        (if-let [cached
                                 (find @dependency-cache [lib id])]
                          (val cached)
                          (let [value (vec (coord-deps lib coordinate))]
                            (swap! dependency-cache assoc [lib id] value)
                            value)))
                      queue
                      (if child-predicate
                        (conj queue
                              {:grenadine.expander/children true
                               :children children
                               :parent-path use-path
                               :child-predicate child-predicate})
                        queue)
                      trace
                      (if trace?
                        (conj trace
                              {:path (vec parents)
                               :lib lib
                               :coord coordinate
                               :orig-coord original-coordinate
                               :coord-id id
                               :include (boolean include?)
                               :reason reason})
                        trace)]
                  (recur pending queue index version-map
                         (:exclusions update) (:cuts update)
                         (if include? (conj order lib) order)
                         trace)))))
          (let [version-map
                (reduce
                 (fn [result [lib entry]]
                   (if (:select entry) result (dissoc result lib)))
                 version-map
                 version-map)]
            {:libs (selected-libraries version-map)
             :order (vec (distinct order))
             :warnings @warnings
             :trace (when trace? {:log trace :vmap version-map})}))))))
