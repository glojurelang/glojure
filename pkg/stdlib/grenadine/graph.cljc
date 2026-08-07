(ns grenadine.graph
  "Pure path-aware dependency graph traversal and version mediation."
  (:require [grenadine.coordinate :as coordinate]
            [grenadine.expander :as expander]
            [grenadine.version :as version]))

(defn- split-lib
  [lib]
  (coordinate/split-lib lib))

(defn- ga
  [{:keys [group artifact classifier]}]
  (coordinate/lib-key group artifact classifier))

(defn- base-ga
  [{:keys [group artifact]}]
  [group artifact])

(defn- gav
  [{:keys [version] :as coordinate}]
  (conj (ga coordinate) version))

(defn- coord-order
  [{:keys [version] :as coordinate}]
  (conj (ga coordinate) version))

(defn- normalize-exclusions
  [exclusions]
  (set
   (map
    (fn [exclusion]
      (if (map? exclusion)
        (base-ga exclusion)
        (let [[group artifact] (split-lib exclusion)]
          [group artifact])))
    exclusions)))

(defn- root-coordinates
  [deps]
  (->> deps
       (map
        (fn [entry]
          (let [[lib coordinate] entry
                [group artifact classifier] (split-lib lib)
                version (:mvn/version coordinate)]
            (when-not version
              (throw
               (ex-info (str "Dependency requires :mvn/version: " lib)
                        {:type :grenadine.graph/unsupported-coordinate
                         :lib lib
                         :coordinate coordinate})))
            {:coords (cond-> {:group group :artifact artifact :version version}
                       classifier (assoc :classifier classifier))
             :edge-exclusions
             (normalize-exclusions (:exclusions coordinate #{}))})))
       (sort-by #(coord-order (:coords %)))
       vec))

(defn- kept-scope?
  [scope]
  (or (nil? scope)
      (= scope "compile")
      (= scope "runtime")))

(defn- occurrence
  [state order]
  {:coords (:coords state)
   :depth (:depth state)
   :order order
   :via (:via state)
   :scope (or (:scope state) "compile")
   :direct? (zero? (:depth state))})

(defn- omitted
  [state reason]
  {:coords (:coords state)
   :via (:via state)
   :reason reason})

(defn- enumerate
  [deps {:keys [pom-fn include-optional? exclusions]}]
  (when-not pom-fn
    (throw (ex-info "resolve-graph requires :pom-fn"
                    {:type :grenadine.graph/missing-pom-fn})))
  (let [global-exclusions (normalize-exclusions exclusions)
        initial
        (mapv
         (fn [{:keys [coords edge-exclusions]}]
           {:coords coords
            :depth 0
            :via [coords]
            :blocked global-exclusions
            :edge-exclusions edge-exclusions
            :scope "compile"})
         (root-coordinates deps))]
    (loop [queue initial
           index 0
           order 0
           occurrences []
           graph {}
           omitted-items []
           warnings []]
      (if (= index (count queue))
        {:occurrences occurrences
         :graph graph
         :omitted omitted-items
         :warnings warnings}
        (let [state (nth queue index)
              coords (:coords state)
              path-gavs (set (map gav (butlast (:via state))))]
          (cond
            (contains? (:blocked state) (base-ga coords))
            (recur queue (inc index) order occurrences graph
                   (conj omitted-items (omitted state :excluded))
                   warnings)

            (not (kept-scope? (:scope state)))
            (recur queue (inc index) order occurrences graph
                   (conj omitted-items (omitted state :scope))
                   warnings)

            (and (:optional state) (not include-optional?))
            (recur queue (inc index) order occurrences graph
                   (conj omitted-items (omitted state :optional))
                   warnings)

            (contains? path-gavs (gav coords))
            (recur queue (inc index) order occurrences graph
                   (conj omitted-items (omitted state :cycle))
                   warnings)

            :else
            (let [pom (pom-fn coords)
                  children (:deps pom)
                  child-blocked
                  (into (:blocked state) (:edge-exclusions state))
                  next-states
                  (mapv
                   (fn [dependency]
                     (let [child-coords
                           (select-keys dependency
                                        [:group :artifact :classifier :version])]
                       {:coords child-coords
                        :depth (inc (:depth state))
                        :via (conj (:via state) child-coords)
                        :blocked child-blocked
                        :edge-exclusions
                        (normalize-exclusions (:exclusions dependency #{}))
                        :scope (:scope dependency)
                        :optional (:optional dependency)}))
                   children)]
              (recur
               (into queue next-states)
               (inc index)
               (inc order)
               (conj occurrences (occurrence state order))
               (assoc graph (gav coords) (mapv :coords next-states))
               omitted-items
               warnings))))))))

(defn- nearer
  [left right]
  (if (neg? (compare [(:depth left) (:order left)]
                     [(:depth right) (:order right)]))
    left
    right))

(defn- newer
  [left right]
  (let [comparison
        (version/compare-versions
         (get-in left [:coords :version])
         (get-in right [:coords :version]))]
    (cond
      (pos? comparison) left
      (neg? comparison) right
      (< (:order left) (:order right)) left
      :else right)))

(defn- choose
  [mode candidates]
  (case mode
    :nearest (reduce nearer candidates)
    :newest (reduce newer candidates)
    :tools-deps
    (let [direct (filter :direct? candidates)]
      (if (seq direct)
        (reduce nearer direct)
        (reduce newer candidates)))
    (throw (ex-info (str "Unknown mediation mode: " mode)
                    {:type :grenadine.graph/invalid-mediation
                     :mediation mode}))))

(defn- select-occurrences
  [mode occurrences]
  (into {}
        (map
         (fn [entry]
           [(key entry) (choose mode (val entry))])
         (group-by #(ga (:coords %)) occurrences))))

(defn- selected-signature
  [selected]
  (into {}
        (map (fn [entry] [(key entry) (gav (:coords (val entry)))])
             selected)))

(defn- active-occurrence?
  [selected candidate]
  (every?
   (fn [coords]
     (= (gav coords)
        (some-> (get selected (ga coords)) :coords gav)))
   (:via candidate)))

(defn- mediate
  [mode occurrences]
  (loop [selected (select-occurrences mode occurrences)]
    (let [active (filter #(active-occurrence? selected %) occurrences)
          next-selected (select-occurrences mode active)]
      (if (= (selected-signature selected)
             (selected-signature next-selected))
        {:selected next-selected :active (vec active)}
        (recur next-selected)))))

(defn- tools-coordinate
  [coordinate]
  (cond-> (select-keys coordinate [:group :artifact :classifier :version])
    (seq (:exclusions coordinate))
    (assoc :exclusions (normalize-exclusions (:exclusions coordinate)))))

(defn- tools-deps-expansion
  [deps {:keys [pom-fn include-optional? exclusions]}]
  (let [global-exclusions (normalize-exclusions exclusions)
        roots
        (->> (root-coordinates deps)
             (remove #(contains? global-exclusions (base-ga (:coords %))))
             (mapv
              (fn [{:keys [coords edge-exclusions]}]
                [(ga coords)
                 (cond-> coords
                   (seq edge-exclusions)
                   (assoc :exclusions edge-exclusions))])))]
    (expander/expand-deps
     roots
     {:coord-id (fn [_ coordinate] (gav coordinate))
      :known-coordinate?
      (fn [coordinate]
        (and (map? coordinate) (seq (:version coordinate))))
      :compare-versions
      (fn [_ left right]
        (version/compare-versions (:version left) (:version right)))
      :coord-deps
      (fn [_ coordinate]
        (->> (:deps (pom-fn coordinate))
             (filter #(and (kept-scope? (:scope %))
                           (or include-optional? (not (:optional %)))
                           (not (contains? global-exclusions (base-ga %)))))
             (mapv (fn [dependency]
                     [(ga dependency) (tools-coordinate dependency)]))))})))

(defn- expanded-selection
  [occurrences libraries]
  (into
   {}
   (map
    (fn [[key coordinate]]
      (let [matches
            (filter #(= (gav coordinate) (gav (:coords %))) occurrences)
            selected
            (if (seq matches)
              (reduce nearer matches)
              {:coords (select-keys coordinate
                                    [:group :artifact :classifier :version])
               :depth 0
               :order 0
               :via [(select-keys coordinate
                                  [:group :artifact :classifier :version])]
               :scope "compile"
               :direct? true})]
        [key selected]))
    libraries)))

(defn- tools-deps-mediate
  [occurrences expansion]
  (let [selected (expanded-selection occurrences (:libs expansion))]
    {:selected selected
     :active (vec (filter #(active-occurrence? selected %) occurrences))
     :warnings (:warnings expansion)}))

(defn resolve-graph
  "Resolve Maven dependency occurrences and mediate versions.

  `deps` is a deps.edn-style dependency map. `:pom-fn` must return an effective
  POM. The result's `:selected` map is keyed by `[group artifact]`, where a
  classifier is represented as `artifact$classifier`."
  [deps opts]
  (let [enumerated (enumerate deps opts)
        mode (:mediation opts :newest)
        mediated
        (if (= mode :tools-deps)
          (tools-deps-mediate
           (:occurrences enumerated)
           (tools-deps-expansion deps opts))
          (mediate mode (:occurrences enumerated)))
        selected (:selected mediated)
        active (:active mediated)
        grouped (group-by #(ga (:coords %)) (:occurrences enumerated))
        conflicts
        (mapcat
         (fn [entry]
           (let [key (key entry)
                 winner (get selected key)]
             (->> (val entry)
                  (remove #(= (gav (:coords %)) (gav (:coords winner))))
                  (map #(assoc (select-keys % [:coords :via])
                               :reason :version-conflict)))))
         grouped)]
    {:selected selected
     :graph
     (into {}
           (filter
            (fn [entry]
              (let [version (peek (key entry))
                    lib-key (pop (key entry))]
                (= version
                   (get-in selected
                           [lib-key :coords :version])))))
           (:graph enumerated))
     :omitted (vec (concat (:omitted enumerated) conflicts))
     :warnings (vec (concat (:warnings enumerated)
                            (:warnings mediated)))
     :occurrences active}))
