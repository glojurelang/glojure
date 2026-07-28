(ns benchmark.suite.persistent-map)

(defn initial-map [size]
  (loop [i 0
         values {}]
    (if (= i size)
      values
      (recur (inc i)
             (assoc values i (+ (* i 17) 3))))))

(defn update-map [values size round]
  (loop [i 0
         result values]
    (if (= i size)
      result
      (let [key (mod (+ (* i 97) (* round 53)) size)]
        (recur (inc i)
               (assoc result key (+ (get result key) round i)))))))

(defn remove-some [values size]
  (loop [key 0
         result values]
    (if (>= key size)
      result
      (recur (+ key 7) (dissoc result key)))))

(defn checksum [values size]
  (loop [i 0
         total 0]
    (if (= i size)
      total
      (recur (inc i)
             (+ total (get values i 0))))))

(defn run []
  (let [size 30000
        updated
        (loop [round 1
               values (initial-map size)]
          (if (> round 6)
            values
            (recur (inc round) (update-map values size round))))
        final-values (remove-some updated size)]
    [(count final-values)
     (checksum final-values size)
     (get final-values 12345)
     (contains? final-values 12348)]))
