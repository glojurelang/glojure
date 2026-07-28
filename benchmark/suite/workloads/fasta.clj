(ns benchmark.suite.fasta)

(def alphabet "ACGT")

(defn next-random [state]
  (mod (+ (* state 3877) 29573) 139968))

(defn sequence-chunk [state length]
  (loop [i 0
         random state
         characters []]
    (if (= i length)
      [(apply str characters) random]
      (let [next (next-random random)]
        (recur (inc i)
               next
               (conj characters
                     (nth alphabet (mod next (count alphabet)))))))))

(defn generate-sequence [length]
  (loop [remaining length
         random 42
         chunks []]
    (if (zero? remaining)
      (apply str chunks)
      (let [chunk-length (min 60 remaining)
            [chunk next-random] (sequence-chunk random chunk-length)]
        (recur (- remaining chunk-length)
               next-random
               (conj chunks chunk))))))

(defn weighted-checksum [sequence]
  (loop [i 0
         checksum 0]
    (if (= i (count sequence))
      checksum
      (recur (inc i)
             (+ checksum
                (* (inc i)
                   (case (nth sequence i)
                     \A 1
                     \C 2
                     \G 3
                     \T 4)))))))

(defn run []
  (let [sequence (generate-sequence 180000)
        counts (frequencies sequence)]
    [(count sequence)
     [(get counts \A) (get counts \C) (get counts \G) (get counts \T)]
     (weighted-checksum sequence)]))
