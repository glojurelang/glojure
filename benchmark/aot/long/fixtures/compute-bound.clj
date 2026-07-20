(ns bench.long.compute-bound)

;; This is let-go's docs/perf/microbench/compute-bound.lg workload, expressed
;; as a function so both AOT compilers can expose the same entry point.
(defn run []
  (reduce
    (fn [acc _]
      (+ acc
         (reduce +
                 0
                 (map (fn [x] (* x x))
                      (filter even? (range 50000))))))
    0
    (range 60)))
