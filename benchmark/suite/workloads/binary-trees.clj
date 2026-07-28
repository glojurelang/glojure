(ns benchmark.suite.binary-trees)

(defrecord Tree [item left right])

(defn make-tree [item depth]
  (if (zero? depth)
    (->Tree item nil nil)
    (let [child-depth (dec depth)
          child-item (* item 2)]
      (->Tree item
              (make-tree (dec child-item) child-depth)
              (make-tree child-item child-depth)))))

(defn check-tree [tree]
  (if (nil? (:left tree))
    (:item tree)
    (+ (:item tree)
       (check-tree (:left tree))
       (- (check-tree (:right tree))))))

(defn power-of-two [exponent]
  (loop [n exponent
         value 1]
    (if (zero? n)
      value
      (recur (dec n) (* value 2)))))

(defn depth-check [max-depth min-depth depth]
  (let [iterations (power-of-two (+ max-depth min-depth (- depth)))]
    (loop [i 1
           total 0]
      (if (> i iterations)
        [depth iterations total]
        (recur (inc i)
               (+ total
                  (check-tree (make-tree i depth))
                  (check-tree (make-tree (- i) depth))))))))

(defn run []
  (let [min-depth 4
        max-depth 12
        stretch-depth (inc max-depth)
        stretch-check (check-tree (make-tree 0 stretch-depth))
        long-lived-tree (make-tree 0 max-depth)]
    [stretch-check
     (loop [depth min-depth
            checks []]
       (if (> depth max-depth)
         checks
         (recur (+ depth 2)
                (conj checks
                      (depth-check max-depth min-depth depth)))))
     (check-tree long-lived-tree)]))
