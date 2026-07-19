(def width 120)
(def height 72)
(def max-iterations 60)
(def scale 1000)

(defn escape-count [cx cy]
  (loop [zr 0 zi 0 iteration 0]
    (let [zr2 (quot (* zr zr) scale)
          zi2 (quot (* zi zi) scale)]
      (if (or (= iteration max-iterations)
              (> (+ zr2 zi2) (* 4 scale)))
        iteration
        (recur (+ (- zr2 zi2) cx)
               (+ (quot (* 2 zr zi) scale) cy)
               (inc iteration))))))

(println
  (loop [y 0 checksum 0]
    (if (= y height)
      checksum
      (let [cy (+ -1000 (quot (* y 2000) height))
            row-sum
            (loop [x 0 total 0]
              (if (= x width)
                total
                (let [cx (+ -2500 (quot (* x 3500) width))]
                  (recur (inc x) (+ total (escape-count cx cy))))))]
        (recur (inc y) (+ checksum (* (inc y) row-sum)))))))
