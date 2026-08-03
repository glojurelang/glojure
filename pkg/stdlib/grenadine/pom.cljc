(ns grenadine.pom
  "Pure Maven POM normalization and effective-model construction."
  (:require [clojure.string :as str]
            [grenadine.xml :as xml]))

(defn- elements
  [node tag]
  (filter #(and (map? %) (= tag (:tag %))) (:content node)))

(defn- element
  [node tag]
  (first (elements node tag)))

(defn- text
  [node]
  (when node
    (let [value (str/trim
                 (apply str (filter string? (:content node))))]
      (when (seq value) value))))

(defn- field
  [node tag]
  (text (element node tag)))

(defn- present
  [m key value]
  (if (nil? value) m (assoc m key value)))

(defn- parse-exclusion
  [node]
  {:group (field node :groupId)
   :artifact (field node :artifactId)})

(defn- parse-dependency
  [node]
  (let [exclusions-node (element node :exclusions)
        exclusions
        (if exclusions-node
          (->> (elements exclusions-node :exclusion)
               (map parse-exclusion)
               (remove #(or (nil? (:group %)) (nil? (:artifact %))))
               set)
          #{})]
    (-> {:group (field node :groupId)
         :artifact (field node :artifactId)
         :exclusions exclusions}
        (present :version (field node :version))
        (present :scope (field node :scope))
        (present :type (field node :type))
        (present :classifier (field node :classifier))
        (present :optional
                 (when-let [value (field node :optional)]
                   (= "true" (str/lower-case value)))))))

(defn- parse-dependencies
  [node]
  (if node
    (mapv parse-dependency (elements node :dependency))
    []))

(defn- parse-properties
  [node]
  (if node
    (reduce
     (fn [result child]
       (if (map? child)
         (assoc result (name (:tag child)) (or (text child) ""))
         result))
     {}
     (:content node))
    {}))

(defn parse-pom
  "Parse POM XML into a canonical raw model. No inheritance or interpolation is
  performed here."
  [xml-source]
  (let [project (xml/parse xml-source)]
    (when (not= :project (:tag project))
      (throw (ex-info "POM document root must be <project>"
                      {:type :grenadine.pom/not-project
                       :tag (:tag project)})))
    (let [parent-node (element project :parent)
          parent
          (when parent-node
            {:group (field parent-node :groupId)
             :artifact (field parent-node :artifactId)
             :version (field parent-node :version)})
          management-node
          (some-> (element project :dependencyManagement)
                  (element :dependencies))]
      {:model-version (field project :modelVersion)
       :declared-coords
       {:group (field project :groupId)
        :artifact (field project :artifactId)
        :version (field project :version)}
       :parent parent
       :packaging (or (field project :packaging) "jar")
       :properties (parse-properties (element project :properties))
       :dependency-management (parse-dependencies management-node)
       :dependencies
       (parse-dependencies (element project :dependencies))})))

(defn- coords-key
  [{:keys [group artifact version]}]
  (str group "/" artifact "@" version))

(defn- unresolved-property?
  [value]
  (and (string? value) (str/includes? value "${")))

(defn- find-property-end
  [value offset]
  (loop [i offset]
    (cond
      (= i (count value)) nil
      (= \} (nth value i)) i
      :else (recur (inc i)))))

(declare interpolate-value)

(defn- property-value
  [properties key trail]
  (if (some #(= key %) trail)
    (throw
     (ex-info (str "Property interpolation cycle: "
                   (str/join " -> " (conj trail key)))
              {:type :grenadine.pom/property-cycle
               :properties (conj trail key)}))
    (when-let [value (get properties key)]
      (interpolate-value value properties (conj trail key)))))

(defn- interpolate-value
  [value properties trail]
  (if (not (string? value))
    value
    (loop [i 0 piece-start 0 pieces []]
      (if (= i (count value))
        (apply str (conj pieces (subs value piece-start)))
        (if (and (= \$ (nth value i))
                 (< (inc i) (count value))
                 (= \{ (nth value (inc i))))
          (if-let [end (find-property-end value (+ i 2))]
            (let [key (subs value (+ i 2) end)
                  replacement (property-value properties key trail)]
              (recur (inc end)
                     (inc end)
                     (conj pieces
                           (subs value piece-start i)
                           (if (nil? replacement)
                             (subs value i (inc end))
                             replacement))))
            (recur (inc i) piece-start pieces))
          (recur (inc i) piece-start pieces))))))

(defn interpolate-string
  "Interpolate Maven-style `${name}` references in a string. Unknown
  properties remain visible so callers can distinguish them from empty values."
  [value properties]
  (interpolate-value value properties []))

(defn- interpolate-map
  [m properties]
  (reduce
   (fn [result entry]
     (let [key (key entry)
           value (val entry)]
     (assoc result key
            (cond
              (string? value) (interpolate-string value properties)
              (set? value) (set (map #(interpolate-map % properties) value))
              :else value))))
   {}
   m))

(defn- dep-key
  [{:keys [group artifact type classifier]}]
  [group artifact (or type "jar") classifier])

(defn- merge-ordered
  "Merge ordered dependency vectors by Maven identity. A later declaration
  replaces the earlier value without moving its position."
  [base additions]
  (reduce
   (fn [result dependency]
     (let [key (dep-key dependency)
           index (first
                  (keep-indexed
                   (fn [i existing]
                     (when (= key (dep-key existing)) i))
                   result))]
       (if (nil? index)
         (conj result dependency)
         (assoc result index dependency))))
   (vec base)
   additions))

(defn- builtin-properties
  [coords parent packaging]
  (merge
   {"project.groupId" (:group coords)
    "pom.groupId" (:group coords)
    "groupId" (:group coords)
    "project.artifactId" (:artifact coords)
    "pom.artifactId" (:artifact coords)
    "artifactId" (:artifact coords)
    "project.version" (:version coords)
    "pom.version" (:version coords)
    "version" (:version coords)
    "project.packaging" packaging
    "pom.packaging" packaging}
   (when parent
     {"project.parent.groupId" (get-in parent [:coords :group])
      "project.parent.artifactId" (get-in parent [:coords :artifact])
      "project.parent.version" (get-in parent [:coords :version])
      "parent.groupId" (get-in parent [:coords :group])
      "parent.artifactId" (get-in parent [:coords :artifact])
      "parent.version" (get-in parent [:coords :version])})))

(defn- assert-resolved-coordinate
  [context dependency]
  (doseq [key [:group :artifact :version]]
    (let [value (get dependency key)]
      (when (or (nil? value) (unresolved-property? value))
        (throw
         (ex-info (str "Unresolved " (name key) " in " context)
                  {:type :grenadine.pom/unresolved-coordinate
                   :context context
                   :field key
                   :dependency dependency})))))
  dependency)

(declare effective-pom*)

(defn- import-boms
  [management fetch-pom path]
  (reduce
   (fn [result dependency]
     (if (and (= "import" (:scope dependency))
              (= "pom" (or (:type dependency) "jar")))
       (let [coords (select-keys dependency [:group :artifact :version])
             bom (effective-pom* coords fetch-pom path)]
         (merge-ordered result (vals (:dep-management bom))))
       result))
   []
   management))

(defn- apply-management
  [management dependency]
  (if-let [managed (get management (dep-key dependency))]
    (let [explicit-exclusions (:exclusions dependency)
          merged (merge managed dependency)]
      (assoc merged :exclusions
             (if (seq explicit-exclusions)
               explicit-exclusions
               (:exclusions managed #{}))))
    dependency))

(defn- effective-pom*
  [coords fetch-pom path]
  (let [identity (coords-key coords)]
    (when (some #(= identity %) path)
      (throw
       (ex-info (str "POM parent/BOM cycle: "
                     (str/join " -> " (conj path identity)))
                {:type :grenadine.pom/model-cycle
                 :coordinates (conj path identity)})))
    (let [raw-source (fetch-pom coords)
          raw (if (string? raw-source) (parse-pom raw-source) raw-source)]
      (when-not raw
        (throw (ex-info (str "POM not found: " identity)
                        {:type :grenadine.pom/not-found
                         :coords coords})))
      (let [next-path (conj path identity)
            parent-coords (:parent raw)
            parent (when parent-coords
                     (effective-pom*
                      (assert-resolved-coordinate "parent" parent-coords)
                      fetch-pom
                      next-path))
            declared (:declared-coords raw)
            preliminary-coords
            {:group (or (:group declared) (get-in parent [:coords :group]))
             :artifact (:artifact declared)
             :version (or (:version declared) (get-in parent [:coords :version]))}
            inherited-properties (or (:properties parent) {})
            raw-properties (merge inherited-properties (:properties raw))
            properties-with-builtins
            (merge raw-properties
                   (builtin-properties preliminary-coords
                                       parent
                                       (:packaging raw)))
            coords
            (assert-resolved-coordinate
             "project"
             (interpolate-map preliminary-coords properties-with-builtins))
            all-properties
            (merge raw-properties
                   (builtin-properties coords parent (:packaging raw)))
            properties
            (reduce
             (fn [result entry]
               (assoc result
                      (key entry)
                      (interpolate-string (val entry) all-properties)))
             {}
             raw-properties)
            interpolation-context
            (merge properties (builtin-properties coords parent (:packaging raw)))
            declared-management
            (mapv #(interpolate-map % interpolation-context)
                  (:dependency-management raw))
            imported-management
            (import-boms declared-management fetch-pom next-path)
            parent-management (or (:dep-management parent) {})
            management-entries
            (merge-ordered
             (vals parent-management)
             (concat imported-management
                     (remove #(= "import" (:scope %))
                             declared-management)))
            management
            (into {} (map (fn [dependency]
                            [(dep-key dependency) dependency])
                          management-entries))
            declared-dependencies
            (mapv
             (fn [dependency]
               (->> (interpolate-map dependency interpolation-context)
                    (apply-management management)
                    (assert-resolved-coordinate "dependency")))
             (:dependencies raw))
            dependencies
            (merge-ordered (or (:deps parent) []) declared-dependencies)]
        {:coords coords
         :packaging
         (interpolate-string (:packaging raw) interpolation-context)
         :properties properties
         :dep-management management
         :deps dependencies}))))

(defn effective-pom
  "Build an effective POM for `coords`.

  `fetch-pom` is a pure lookup function from coordinate map to either XML text
  or a canonical raw model returned by `parse-pom`."
  [coords fetch-pom]
  (effective-pom* coords fetch-pom []))
