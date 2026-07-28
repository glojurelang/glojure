(ns benchmark.suite.k-nucleotide)

(def motif "GGTATTTTAATTTATAGTACCGGATGCCCTGCA")

(defn repeat-motif [length]
  (let [motif-length (count motif)]
    (loop [i 0
           chunks []]
      (if (>= i length)
        (subs (apply str chunks) 0 length)
        (recur (+ i motif-length) (conj chunks motif))))))

(defn kmer-frequencies [dna width]
  (let [limit (inc (- (count dna) width))]
    (loop [i 0
           frequencies {}]
      (if (= i limit)
        frequencies
        (let [token (subs dna i (+ i width))]
          (recur (inc i)
                 (assoc frequencies
                        token
                        (inc (get frequencies token 0)))))))))

(defn top-frequency-checksum [frequencies]
  (reduce
    +
    0
    (map-indexed
      (fn [index entry]
        (* (inc index) (nth entry 1)))
      (sort-by
        (fn [entry] [(- (nth entry 1)) (nth entry 0)])
        frequencies))))

(defn run []
  (let [dna (repeat-motif 120000)
        pairs (kmer-frequencies dna 2)
        fragments ["GGT" "GGTA" "GGTATT" "GGTATTTTA"]]
    [(count pairs)
     (top-frequency-checksum pairs)
     (mapv (fn [fragment]
             (get (kmer-frequencies dna (count fragment)) fragment 0))
           fragments)]))
