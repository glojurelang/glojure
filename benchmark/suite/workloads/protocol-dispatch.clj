(ns benchmark.suite.protocol-dispatch)

(defprotocol Accumulator
  (combine [marker left right]))

;; Extending nil keeps the source host-neutral: there is no Java class name or
;; Glojure implementation type in the workload. The accumulator state remains
;; ordinary Clojure data while every combine call still takes the protocol
;; dispatch path.
(extend-protocol Accumulator
  nil
  (combine [_ left right]
    (+ (* left 1.0000001) right)))

(defn run []
  (let [result
        (loop [i 0
               value 0.25
               checksum 0.0]
          (if (= i 500000)
            [value checksum]
            (let [input (* 0.000001 (- (mod i 2000) 1000))
                  next-value (combine nil value input)]
              (recur (inc i)
                     next-value
                     (+ checksum next-value)))))
        value (nth result 0)
        checksum (nth result 1)]
    [(long (* 1000000.0 value))
     (long checksum)]))
