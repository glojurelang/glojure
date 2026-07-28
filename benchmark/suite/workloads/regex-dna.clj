(ns benchmark.suite.regex-dna
  (:require [clojure.string :as string]))

(def motif "agggtaaaTTTACCctagggtaaaGGGTacccta")
(def patterns
  [#"agggtaaa|tttaccct"
   #"[cgt]gggtaaa|tttaccc[acg]"
   #"a[act]ggtaaa|tttacc[agt]t"
   #"ag[act]gtaaa|tttac[agt]ct"
   #"agg[act]taaa|ttta[agt]cct"])

(defn generated-input []
  (apply str
         (map-indexed
           (fn [index value]
             (if (zero? (mod index 17))
               (str ">sequence-" index "\n" value "\n")
               (str value "\n")))
           (repeat 7000 motif))))

(defn count-pattern [sequence pattern]
  (count (re-seq pattern sequence)))

(defn run []
  (let [input (generated-input)
        cleaned (string/replace input #">[^\n]*\n|\n" "")
        substituted (-> cleaned
                        (string/replace #"tHa[Nt]" "<4>")
                        (string/replace #"aND|caN|Ha[DS]|WaS" "<3>")
                        (string/replace #"a[NSt]|BY" "<2>")
                        (string/replace #"<[^>]*>" "|")
                        (string/replace #"\|[^|][^|]*\|" "-"))]
    [(count input)
     (count cleaned)
     (mapv (fn [pattern] (count-pattern cleaned pattern)) patterns)
     (count substituted)]))
