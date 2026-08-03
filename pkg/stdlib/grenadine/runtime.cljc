(ns grenadine.runtime
  "Shared add-only behavior for non-JVM runtime facades."
  (:require [grenadine.core :as grenadine]))

(defn current-basis
  [basis]
  @basis)

(defn- missing-libs
  [basis libs]
  (reduce
   (fn [result [lib coordinate]]
     (if (contains? @basis lib)
       result
       (assoc result lib coordinate)))
   {}
   libs))

(defn- retained-warnings
  [basis libs]
  (->> libs
       (keep
        (fn [[lib requested]]
          (when-let [loaded (get @basis lib)]
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
        (swap! basis merge missing)
        (update result :warnings into retained)))))
