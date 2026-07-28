(ns benchmark.suite.sorting)

(defn generated-values [size]
  (loop [i 0
         random 17
         values []]
    (if (= i size)
      values
      (let [next-random (mod (+ (* random 48271) 31) 2147483647)]
        (recur (inc i)
               next-random
               (conj values (- (mod next-random 2000000) 1000000)))))))

(defn checksum [values]
  (loop [remaining values
         index 0
         total 0]
    (if (empty? remaining)
      total
      (recur (next remaining)
             (inc index)
             (+ total (* (inc index) (first remaining)))))))

(defn run []
  (let [ascending (sort (generated-values 90000))
        descending (sort > (take 30000 ascending))]
    [(first ascending)
     (last ascending)
     (first descending)
     (last descending)
     (checksum (take 2000 ascending))
     (checksum (take 2000 descending))]))
