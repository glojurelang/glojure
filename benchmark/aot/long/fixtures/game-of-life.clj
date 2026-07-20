(ns bench.long.game-of-life)

(defn width [] 48)
(defn height [] 48)
(defn cell-count [] (* (width) (height)))

(defn initial-state []
  (mapv (fn [i]
          (if (< (mod (+ (* i 17) (* i i 3)) 23) 7) 1 0))
        (range (cell-count))))

(defn cell [state x y]
  (nth state (+ (mod x (width)) (* (width) (mod y (height))))))

(defn neighbor-count [state i]
  (let [x (mod i (width))
        y (quot i (width))]
    (+ (cell state (dec x) (dec y))
       (cell state x       (dec y))
       (cell state (inc x) (dec y))
       (cell state (dec x) y)
       (cell state (inc x) y)
       (cell state (dec x) (inc y))
       (cell state x       (inc y))
       (cell state (inc x) (inc y)))))

(defn step [state]
  (mapv (fn [i]
          (let [alive (nth state i)
                n (neighbor-count state i)]
            (if (or (= n 3) (and (= alive 1) (= n 2))) 1 0)))
        (range (cell-count))))

(defn simulate []
  (loop [state (initial-state)
         generation 0]
    (if (= generation 75)
      state
      (recur (step state) (inc generation)))))

(defn checksum [state]
  (loop [i 0
         total 0]
    (if (= i (cell-count))
      total
      (recur (inc i) (+ total (* (inc i) (nth state i)))))))

(defn run []
  (loop [iteration 0
         total 0]
    (if (= iteration 8)
      total
      (recur (inc iteration)
             (+ total (checksum (simulate)))))))
