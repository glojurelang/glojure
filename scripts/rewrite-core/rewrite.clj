(ns glojure-rewrite-core
  (:require [rewrite-clj.zip :as z]))

(def zloc (z/of-string (slurp "./core.clj")))

;; remove until we're at the end of all forms
(defn skip-n [zloc n]
  ;; apply z/right n times
  (let [zloc (nth (iterate z/right zloc) n)]
    (loop [zloc (z/right zloc)]
      (if (z/end? zloc)
        zloc
        (recur (z/next (z/remove zloc)))))))

(def replacements
  [;; replace clojure.core with glojure.core
   [(fn select [zloc] (= 'clojure.core (z/sexpr zloc)))
    (fn visit [zloc] (z/replace zloc 'glojure.core))]
   ;; replace (. clojure.lang.PersistentList creator) with glojure.core.CreateList
   [(fn select [zloc] (= '(. clojure.lang.PersistentList creator) (z/sexpr zloc)))
    (fn visit [zloc] (z/replace zloc 'glojure.lang.CreateList))]
   ;; end
   ])

(defn rewrite-core [zloc]
  (loop [zloc (z/of-node (z/root (skip-n zloc 3)))]
    (if (z/end? zloc)
      (z/root-string zloc)
      (do
        ;; if one of the selectors in replacements matches, replace the form
        (let [zloc (reduce (fn [zloc [select visit]]
                             (if (select zloc)
                               (visit zloc)
                               zloc))
                           zloc
                           replacements)]
          (recur (z/next zloc)))))))

(print (rewrite-core zloc))
