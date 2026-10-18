(ns codegen.test.uuid-literals)

(def ids [#uuid "00000000-0000-0000-0000-000000000000"
          #uuid "f81d4fae-7dec-11d0-a765-00a0c91e6bf6"])

(defn ^{:expected-output
        [[true true]
         ["00000000-0000-0000-0000-000000000000"
          "f81d4fae-7dec-11d0-a765-00a0c91e6bf6"]
         true]} -main []
  [(mapv uuid? ids)
   (mapv str ids)
   (= ids [#uuid "00000000-0000-0000-0000-000000000000"
           #uuid "f81d4fae-7dec-11d0-a765-00a0c91e6bf6"])])
