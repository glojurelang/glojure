(ns bench.long.event-analytics)

(defn services []
  [:api :worker :billing :search :notifications])

(defn statuses []
  [200 200 200 201 204 400 404 429 500 503])

(defn event [i]
  {:service (nth (services) (mod (+ (* i 7) 3) (count (services))))
   :status (nth (statuses) (mod (+ (* i 13) (quot i 17)) (count (statuses))))
   :latency-ms (+ 5 (mod (+ (* i 37) (quot i 11)) 900))
   :bytes (+ 200 (mod (* i 7919) 50000))})

(defn summarize []
  (reduce
    (fn [totals e]
      (let [service (:service e)
            status (:status e)
            failed (if (>= status 400) 1 0)]
        (-> totals
            (update-in [service :requests] (fnil inc 0))
            (update-in [service :failures] (fnil + 0) failed)
            (update-in [service :latency-ms] (fnil + 0) (:latency-ms e))
            (update-in [service :bytes] (fnil + 0) (:bytes e)))))
    {}
    ;; Keep the higher-order boundary boxed in both AOT compilers. let-go
    ;; lowers event itself to an int-specialized direct function, which cannot
    ;; also be installed as an ordinary Var callback.
    (map (fn [i] (event i)) (range 75000))))

(defn checksum [summary]
  (reduce
    (fn [total service]
      (+ total
         (get-in summary [service :requests])
         (get-in summary [service :failures])
         (get-in summary [service :latency-ms])
         (get-in summary [service :bytes])))
    0
    (services)))

(defn run []
  (loop [iteration 0
         total 0]
    (if (= iteration 3)
      total
      (recur (inc iteration)
             (+ total (checksum (summarize)))))))
