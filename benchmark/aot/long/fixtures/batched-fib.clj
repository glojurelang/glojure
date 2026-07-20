(ns bench.long.batched-fib)

(defn fib [n]
  (if (<= n 1)
    n
    (+ (fib (- n 1))
       (fib (- n 2)))))

;; Repeating fib(35), rather than increasing n, lengthens the benchmark without
;; changing its recursion shape or making small input changes exponentially
;; alter the run time.
(defn run []
  (loop [iteration 0
         total 0]
    (if (= iteration 20)
      total
      (recur (inc iteration)
             (+ total (fib 35))))))
