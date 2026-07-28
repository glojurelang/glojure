(ns benchmark.suite.spectral-norm)

(defn matrix-value [i j]
  (/ 1.0
     (+ (quot (* (+ i j) (+ i j 1)) 2)
        i
        1)))

(defn multiply-a [input]
  (let [n (count input)]
    (loop [i 0
           output []]
      (if (= i n)
        output
        (recur
          (inc i)
          (conj output
                (loop [j 0
                       total 0.0]
                  (if (= j n)
                    total
                    (recur (inc j)
                           (+ total
                              (* (matrix-value i j)
                                 (nth input j))))))))))))

(defn multiply-at [input]
  (let [n (count input)]
    (loop [i 0
           output []]
      (if (= i n)
        output
        (recur
          (inc i)
          (conj output
                (loop [j 0
                       total 0.0]
                  (if (= j n)
                    total
                    (recur (inc j)
                           (+ total
                              (* (matrix-value j i)
                                 (nth input j))))))))))))

(defn multiply-ata [input]
  (multiply-at (multiply-a input)))

(defn dot [left right]
  (loop [i 0
         total 0.0]
    (if (= i (count left))
      total
      (recur (inc i)
             (+ total (* (nth left i) (nth right i)))))))

(defn run []
  (let [n 85
        u (vec (repeat n 1.0))
        [u v]
        (loop [iteration 0
               u u
               v u]
          (if (= iteration 10)
            [u v]
            (let [next-v (multiply-ata u)
                  next-u (multiply-ata next-v)]
              (recur (inc iteration) next-u next-v))))
        ratio (/ (dot u v) (dot v v))]
    ;; A scaled integer avoids cross-runtime differences in float printing.
    (long (* 1000000000000.0 ratio))))
