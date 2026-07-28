(ns benchmark.suite.multimethod-dispatch)

(defmulti score
  (fn [event] [(:kind event) (:priority event)]))

(defmethod score [:read :low] [event]
  (+ (:value event) 1))

(defmethod score [:read :high] [event]
  (+ (* (:value event) 2) 3))

(defmethod score [:write :low] [event]
  (- (:value event) 2))

(defmethod score [:write :high] [event]
  (+ (* (:value event) 3) 5))

(defmethod score :default [event]
  (:value event))

(def kinds [:read :write :other])
(def priorities [:low :high])

(defn event [index]
  {:kind (nth kinds (mod index (count kinds)))
   :priority (nth priorities (mod (quot index 3) (count priorities)))
   :value (mod (* index 37) 1000)})

(defn run []
  (loop [index 0
         checksum 0]
    (if (= index 400000)
      checksum
      (recur (inc index)
             (+ checksum (score (event index)))))))
