(def services [:api :worker :billing :search :notifications])
(def statuses [200 200 200 201 204 400 404 429 500 503])
(def event-count 75000)

(defn event [i]
  {:service (nth services (mod (+ (* i 7) 3) (count services)))
   :status (nth statuses (mod (+ (* i 13) (quot i 17)) (count statuses)))
   :latency-ms (+ 5 (mod (+ (* i 37) (quot i 11)) 900))
   :bytes (+ 200 (mod (* i 7919) 50000))})

(def summary
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
    (map event (range event-count))))

(println
  (mapv
    (fn [service]
      [service
       (get-in summary [service :requests])
       (get-in summary [service :failures])
       (get-in summary [service :latency-ms])
       (get-in summary [service :bytes])])
    services))
