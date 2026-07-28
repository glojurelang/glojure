(ns benchmark.suite.prime-workload)

(defn prime? [n]
  (cond
    (< n 2) false
    (= n 2) true
    (even? n) false
    :else
    (loop [divisor 3]
      (cond
        (> (* divisor divisor) n) true
        (zero? (mod n divisor)) false
        :else (recur (+ divisor 2))))))

(defn primes []
  (filter prime? (iterate inc 2)))

(defn prime-factors [n]
  (loop [remaining n
         candidates (primes)
         factors []]
    (let [prime (first candidates)]
      (cond
        (= remaining 1) factors
        (zero? (mod remaining prime))
        (recur (quot remaining prime)
               candidates
               (conj factors prime))
        (> (* prime prime) remaining)
        (conj factors remaining)
        :else
        (recur remaining (next candidates) factors)))))

(defn run []
  (let [inputs (map (fn [i] (+ 1000003 (* i 210))) (range 500))]
    [(reduce + 0 (take 2500 (primes)))
     (reduce + 0 (mapcat prime-factors inputs))]))
