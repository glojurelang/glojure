(ns benchmark.suite.fannkuch-redux)

(defn swap-at [values left right]
  (let [left-value (nth values left)
        right-value (nth values right)]
    (assoc (assoc values left right-value) right left-value)))

(defn reverse-prefix [values length]
  (loop [result values
         left 0
         right (dec length)]
    (if (>= left right)
      result
      (recur (swap-at result left right)
             (inc left)
             (dec right)))))

(defn flip-count [permutation]
  (loop [values permutation
         flips 0]
    (let [first-value (nth values 0)]
      (if (zero? first-value)
        flips
        (recur (reverse-prefix values (inc first-value))
               (inc flips))))))

(defn rotate-prefix-left [values length]
  (let [first-value (nth values 0)]
    (loop [result values
           i 0]
      (if (= i (dec length))
        (assoc result i first-value)
        (recur (assoc result i (nth values (inc i)))
               (inc i))))))

(defn next-permutation [permutation counts n]
  (loop [values (swap-at permutation 0 1)
         counters counts
         r 1]
    (let [next-count (inc (nth counters r))
          next-counters (assoc counters r next-count)]
      (if (<= next-count r)
        [values next-counters]
        (if (= r (dec n))
          nil
          (recur (rotate-prefix-left values (+ r 2))
                 (assoc next-counters r 0)
                 (inc r)))))))

(defn run []
  (let [n 9
        initial (vec (range n))
        counts (vec (repeat n 0))]
    (loop [permutation initial
           counters counts
           index 0
           checksum 0
           maximum 0]
      (let [flips (flip-count permutation)
            signed-flips (if (even? index) flips (- flips))
            next-state (next-permutation permutation counters n)]
        (if (nil? next-state)
          [(+ checksum signed-flips) (max maximum flips)]
          (recur (nth next-state 0)
                 (nth next-state 1)
                 (inc index)
                 (+ checksum signed-flips)
                 (max maximum flips)))))))
