(ns grenadine.runtime
  "Shared add-only behavior for non-JVM runtime facades."
  (:require [grenadine.core :as grenadine]))

(defn current-basis
  [basis]
  (dissoc @basis :grenadine/loaded))

(defn- missing-libs
  [basis libs]
  (reduce
   (fn [result [lib coordinate]]
     (if (contains? (:grenadine/loaded @basis) lib)
       result
       (assoc result lib coordinate)))
   {}
   libs))

(defn- retained-warnings
  [basis libs]
  (->> libs
       (keep
        (fn [[lib requested]]
          (when-let [loaded (get-in @basis [:grenadine/loaded lib])]
            (when (not= loaded requested)
              {:warning :loaded-lib-not-upgraded
               :lib lib
               :loaded loaded
               :requested requested}))))
       vec))

(defn add-libs!
  "Resolve libraries, extract their Clojure sources, and append the roots.

  `add-roots!` is the runtime's load-path mutation hook. `opts` must provide a
  Grenadine host map unless `:install-fn` is supplied for embedding or tests."
  [basis add-roots! libs opts]
  (let [missing (missing-libs basis libs)
        retained (retained-warnings basis libs)]
    (if (empty? missing)
      {:classpath []
       :source-roots []
       :lock nil
       :warnings retained}
      (let [install-fn (or (:install-fn opts) grenadine/install!)
            result (install-fn missing
                               (-> opts
                                   (dissoc :install-fn)
                                   (assoc :source-roots? true
                                          :mediation
                                          (or (:mediation opts)
                                              :tools-deps))))
            roots (vec (:source-roots result))]
        (when (seq roots)
          (add-roots! roots))
        (when-let [new-basis (:basis result)]
          (let [current @basis]
            (swap! basis merge
                   (assoc (dissoc new-basis :libs :classpath :classpath-roots)
                          :libs (merge (:libs current) (:libs new-basis))
                          :classpath (merge (:classpath current)
                                            (:classpath new-basis))
                          :classpath-roots
                          (vec (distinct
                                (concat (:classpath-roots current)
                                        (:classpath-roots new-basis))))))))
        (when-not (:basis result)
          (swap! basis merge {:libs (merge (:libs @basis) missing)}))
        (swap! basis merge
               {:grenadine/loaded
                (merge (:grenadine/loaded @basis) missing)})
        (update result :warnings into retained)))))
