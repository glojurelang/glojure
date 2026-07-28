(ns benchmark.suite.n-body)

(def pairs
  [[0 1] [0 2] [0 3] [0 4]
   [1 2] [1 3] [1 4]
   [2 3] [2 4]
   [3 4]])

;; Bodies are [x y z vx vy vz mass]. Values are the traditional five-body
;; benchmark constants, represented without host arrays or mutable fields.
(def initial-bodies
  [[0.0 0.0 0.0 0.0 0.0 0.0 39.47841760435743]
   [4.841431442464721 -1.1603200440274284 -0.10362204447112311
    0.606326392995832 2.81198684491626 -0.02521836165988763
    0.03769367487038949]
   [8.34336671824458 4.124798564124305 -0.4035234171143214
    -0.41077059553906177 1.7761546290800398 0.008415761376584154
    0.011286326131968767]
   [12.894369562139131 -15.111151401698631 -0.22330757889265573
    0.22621856992079027 1.2634287823489555 -0.0448717377666583
    0.0017237240570597112]
   [15.379697114850917 -25.919314609987964 0.17925877295037118
    0.18263479008236158 1.0057408190378983 -0.034755955504078104
    0.0020336868699246304]])

(defn offset-momentum [bodies]
  (let [[px py pz]
        (reduce
          (fn [[px py pz] body]
            (let [mass (nth body 6)]
              [(+ px (* (nth body 3) mass))
               (+ py (* (nth body 4) mass))
               (+ pz (* (nth body 5) mass))]))
          [0.0 0.0 0.0]
          bodies)
        solar-mass (nth (nth bodies 0) 6)]
    (-> bodies
        (assoc-in [0 3] (/ (- px) solar-mass))
        (assoc-in [0 4] (/ (- py) solar-mass))
        (assoc-in [0 5] (/ (- pz) solar-mass)))))

(defn update-pair [bodies pair dt]
  (let [left-index (nth pair 0)
        right-index (nth pair 1)
        left (nth bodies left-index)
        right (nth bodies right-index)
        dx (- (nth left 0) (nth right 0))
        dy (- (nth left 1) (nth right 1))
        dz (- (nth left 2) (nth right 2))
        distance-squared (+ (* dx dx) (* dy dy) (* dz dz))
        magnitude (/ dt
                     (* distance-squared
                        (Math/sqrt distance-squared)))
        left-mass (nth left 6)
        right-mass (nth right 6)
        left-factor (* right-mass magnitude)
        right-factor (* left-mass magnitude)]
    (-> bodies
        (assoc-in [left-index 3]
                  (- (nth left 3) (* dx left-factor)))
        (assoc-in [left-index 4]
                  (- (nth left 4) (* dy left-factor)))
        (assoc-in [left-index 5]
                  (- (nth left 5) (* dz left-factor)))
        (assoc-in [right-index 3]
                  (+ (nth right 3) (* dx right-factor)))
        (assoc-in [right-index 4]
                  (+ (nth right 4) (* dy right-factor)))
        (assoc-in [right-index 5]
                  (+ (nth right 5) (* dz right-factor))))))

(defn advance [bodies dt]
  (let [with-velocities
        (reduce (fn [state pair] (update-pair state pair dt))
                bodies
                pairs)]
    (mapv
      (fn [body]
        (-> body
            (assoc 0 (+ (nth body 0) (* dt (nth body 3))))
            (assoc 1 (+ (nth body 1) (* dt (nth body 4))))
            (assoc 2 (+ (nth body 2) (* dt (nth body 5))))))
      with-velocities)))

(defn energy [bodies]
  (let [kinetic
        (reduce
          (fn [total body]
            (+ total
               (* 0.5
                  (nth body 6)
                  (+ (* (nth body 3) (nth body 3))
                     (* (nth body 4) (nth body 4))
                     (* (nth body 5) (nth body 5))))))
          0.0
          bodies)]
    (reduce
      (fn [total pair]
        (let [left (nth bodies (nth pair 0))
              right (nth bodies (nth pair 1))
              dx (- (nth left 0) (nth right 0))
              dy (- (nth left 1) (nth right 1))
              dz (- (nth left 2) (nth right 2))
              distance (Math/sqrt (+ (* dx dx) (* dy dy) (* dz dz)))]
          (- total (/ (* (nth left 6) (nth right 6)) distance))))
      kinetic
      pairs)))

(defn run []
  (let [start (offset-momentum initial-bodies)
        finish
        (loop [iteration 0
               bodies start]
          (if (= iteration 3000)
            bodies
            (recur (inc iteration) (advance bodies 0.01))))]
    [(long (* 1000000000.0 (energy start)))
     (long (* 1000000000.0 (energy finish)))]))
