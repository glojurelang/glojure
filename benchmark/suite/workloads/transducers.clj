(ns benchmark.suite.transducers)

(def pipeline
  (comp
    (map inc)
    (filter odd?)
    (map (fn [value] (* value value)))
    (take 50000)))

(defn run []
  [(transduce pipeline + 0 (range 500000))
   (transduce
     (comp
       (filter (fn [value] (zero? (mod value 3))))
       (map (fn [value] (+ value 7))))
     +
     0
     (range 200000))])
