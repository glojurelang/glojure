(ns glojure-rewrite-core
  (:require [rewrite-clj.zip :as z]
            [clojure.string :as s]))

(def zloc (z/of-string (slurp "./core.clj")))

;; remove until we're at the end of all forms
(defn skip-n [zloc n]
  ;; apply z/right n times
  (let [zloc (nth (iterate z/right zloc) n)]
    (loop [zloc (z/right zloc)]
      (if (z/end? zloc)
        zloc
        (recur (z/next (z/remove zloc)))))))

(defn sexpr-replace [old new]
  [(fn select [zloc] (and (z/sexpr-able? zloc) (= old (z/sexpr zloc))))
   (fn visit [zloc] (z/replace zloc new))])

(defn RT-replace
  "Replace all instances of a call to a clojure.lang.RT method fsym with
  the result of calling newfn with the argument forms."
  [fsym newfn]
  [(fn select [zloc] (and (z/list? zloc)
                          (let [sexpr (z/sexpr zloc)]
                            (and (= '. (first sexpr))
                                 (= 'clojure.lang.RT (second sexpr))
                                 (list? (nth sexpr 2))
                                 (= fsym (first (nth sexpr 2)))))))
   (fn visit [zloc] (z/replace zloc (newfn (rest (nth (z/sexpr zloc) 2)))))])

(def replacements
  [
   (sexpr-replace 'clojure.core 'glojure.core)
   (sexpr-replace '(. clojure.lang.PersistentList creator) 'glojure.lang.NewList)
   (sexpr-replace '(setMacro) '(SetMacro))
   (sexpr-replace 'clojure.lang.Symbol 'glojure.lang.Symbol)
   ;; instance? replacements
   (sexpr-replace "Evaluates x and tests if it is an instance of the class\n    c. Returns true or false"
                  "Evaluates x and tests if it is an instance of the type\n    t. Returns true or false")
   (sexpr-replace '(fn instance? [^Class c x] (. c (isInstance x)))
                  '(fn instance? [t x] (glojure.lang.HasType t x)))
   ;;
   (sexpr-replace 'IllegalArgumentException. 'errors.New)
   ;; replace .withMeta
   [(fn select [zloc] (and (z/list? zloc) (= '.withMeta (first (z/sexpr zloc)))))
    (fn visit [zloc] (z/replace zloc
                                `(let* [~'res (glojure.lang.WithMeta ~@(rest (z/sexpr zloc)))]
                                   (if (~'res 1)
                                     (throw (~'res 1))
                                     (~'res 0)))))]

   (RT-replace 'cons #(cons 'glojure.lang.NewCons %))
   (RT-replace 'first #(cons 'glojure.lang.First %))
   (RT-replace 'next #(cons 'glojure.lang.Next %))
   (RT-replace 'more #(cons 'glojure.lang.Rest %))

   [(fn select [zloc] (try
                        (and (symbol? (z/sexpr zloc))
                             (or
                              (and (z/leftmost? zloc) (= 'glojure.lang.RT (-> zloc z/up z/left z/sexpr)))
                              (= 'glojure.lang.RT (-> zloc z/left z/sexpr))))
                        (catch Exception e false)))
    (fn visit [zloc] (z/replace zloc
                                (let [sym (-> zloc z/sexpr str)]
                                  (symbol (str (s/upper-case (first sym)) (subs sym 1))))))]

   (sexpr-replace '.meta '.Meta)
   (sexpr-replace 'clojure.lang.IPersistentMap 'glojure.lang.IPersistentMap)
   (sexpr-replace 'clojure.lang.IPersistentVector 'glojure.lang.IPersistentVector)
   (sexpr-replace 'String 'string)
   (sexpr-replace 'clojure.lang.IMeta 'glojure.lang.IMeta)
   (sexpr-replace 'clojure.lang.RT/conj 'glojure.lang.Conj)
   (sexpr-replace 'withMeta 'WithMeta)

   ;; no need for a special name, as go doesn't have a
   ;; builtin "Equals"
   (sexpr-replace 'clojure.lang.Util/equiv 'glojure.lang.Equal)
   (sexpr-replace 'clojure.lang.Util/equals 'glojure.lang.Equal) ;; TODO: implement both equals and equiv for go!!!
   (sexpr-replace '(. x (meta)) '(.Meta x))

   (sexpr-replace 'clojure.lang.Symbol/intern 'glojure.lang.NewSymbol)
   (sexpr-replace '(clojure.lang.Symbol/intern ns name) '(glojure.lang.InternSymbol ns name))

   (sexpr-replace '.getName '.Name)
   (sexpr-replace '.concat 'glojure.lang.ConcatStrings)
   (sexpr-replace 'clojure.lang.RT/assoc 'glojure.lang.Assoc)
   (sexpr-replace 'clojure.lang.RT/subvec 'glojure.lang.Subvec)
   (sexpr-replace 'clojure.lang.Util/identical 'glojure.lang.Identical)
   (sexpr-replace 'clojure.lang.LazilyPersistentVector/create 'glojure.lang.NewVectorFromCollection)
   (sexpr-replace '(. clojure.lang.RT (seq coll)) '(glojure.lang.Seq coll))
   (sexpr-replace '(list 'new 'clojure.lang.LazySeq (list* '^{:once true} fn* [] body))
                  '(list 'glojure.lang.NewLazySeq (list* '^{:once true} fn* [] body)))
   (sexpr-replace 'clojure.lang.RT/count 'glojure.lang.Count)
   (sexpr-replace 'clojure.lang.IChunkedSeq 'glojure.lang.IChunkedSeq)

   ;; replace (. <fn-form> (applyTo <args>)) with (glojure.lang.Apply <fn-form> <args>)
   [(fn select [zloc] (and (z/list? zloc)
                           (let [sexpr (z/sexpr zloc)]
                             (and
                              (= 3 (count sexpr))
                              (= '. (first sexpr))
                              (list? (nth sexpr 2))
                              (= 'applyTo (first (nth sexpr 2)))))))
    (fn visit [zloc] (z/replace zloc
                                (let [sexpr (z/sexpr zloc)]
                                  `(glojure.lang.Apply ~(nth sexpr 1)
                                                       ~(nth (nth sexpr 2) 1)))))]

   (sexpr-replace '(. clojure.lang.RT (get map key)) '(glojure.lang.Get map key))
   (sexpr-replace '(. clojure.lang.RT (get map key not-found)) '(glojure.lang.GetDefault map key not-found))

   ;; TODO: replace these using the RT-replace function!
   (sexpr-replace '(. clojure.lang.RT (keys map)) '(glojure.lang.Keys map))
   (sexpr-replace '(. clojure.lang.RT (vals map)) '(glojure.lang.Vals map))
   (sexpr-replace '(. clojure.lang.RT (seq map)) '(glojure.lang.Seq map))

   (sexpr-replace '(disjoin key) '(Disjoin key))
   (sexpr-replace
    '((fn [^StringBuilder sb more]
        (if more
          (recur (. sb  (append (str (first more)))) (next more))
          (str sb)))
      (new StringBuilder (str x)) ys)
    '((fn [^strings.Builder sb xs]
        (if xs
          (recur (do (. sb  (WriteString (str (first xs))))
                     sb)
                 (next xs))
          (.String sb)))
      (new strings.Builder) (cons x ys)))
   (sexpr-replace '(. x (toString)) '(glojure.lang.ToString x))
   (sexpr-replace 'getName 'Name)
   (sexpr-replace 'getNamespace 'Namespace)
   (sexpr-replace '.hasRoot '.HasRoot)


   ;; Multi-methods
   [(fn select [zloc] (and (z/list? zloc)
                           (let [sexpr (z/sexpr zloc)]
                             (and
                              (= 'new (first sexpr))
                              (= 'clojure.lang.MultiFn (second sexpr))))))
    (fn visit [zloc] (-> zloc
                         z/down
                         (z/replace 'glojure.lang.NewMultiFn)
                         z/right
                         z/remove))]
   (sexpr-replace 'clojure.lang.MultiFn 'glojure.lang.MultiFn)

   (sexpr-replace '(System/getProperty "line.separator") '"\\n")
   (sexpr-replace 'clojure.lang.ISeq 'glojure.lang.ISeq)
   (sexpr-replace 'clojure.lang.IEditableCollection 'glojure.lang.IEditableCollection)
   (sexpr-replace 'clojure.core/import* 'glojure.lang.Import)

   ;; number checksclasses
   (sexpr-replace '(defn integer?
                     "Returns true if n is an integer"
                     {:added "1.0"
                      :static true}
                     [n]
                     (or (instance? Integer n)
                         (instance? Long n)
                         (instance? clojure.lang.BigInt n)
                         (instance? BigInteger n)
                         (instance? Short n)
                         (instance? Byte n)))
                  '(defn integer?
                     "Returns true if n is an integer"
                     {:added "1.0"
                      :static true}
                     [n]
                     (glojure.lang.IsInteger n))
                  )
   (sexpr-replace 'clojure.lang.RT/uncheckedLongCast 'glojure.lang.AsInt64)
   [(fn select [zloc] (try
                        (and (symbol? (z/sexpr zloc))
                             (or
                              (and (z/leftmost? zloc) (= 'glojure.lang.Numbers (-> zloc z/up z/left z/sexpr)))
                              (= 'glojure.lang.Numbers (-> zloc z/left z/sexpr))))
                        (catch Exception e false)))
    (fn visit [zloc] (z/replace zloc
                                (let [sym (-> zloc z/sexpr str)]
                                  (symbol (str (s/upper-case (first sym)) (subs sym 1))))))]
   (sexpr-replace 'clojure.lang.Numbers 'glojure.lang.Numbers)

   (sexpr-replace 'clojure.core/cond 'glojure.core/cond)

   (sexpr-replace 'clojure.lang.Keyword 'glojure.lang.Keyword)

   (sexpr-replace 'clojure.lang.RT 'glojure.lang.RT)
   (sexpr-replace '(nextID) '(NextID))

   (sexpr-replace '(nth coll index not-found) '(NthDefault coll index not-found))

   [(fn select [zloc] (and (z/list? zloc)
                           (= '. (first (z/sexpr zloc)))
                           (= 'clojure.lang.Symbol (second (z/sexpr zloc)))
                           (= 'intern (first (nth (z/sexpr zloc) 2)))
                           ))
    (fn visit [zloc] (z/replace zloc `(glojure.lang.NewSymbol ~@(rest (nth (z/sexpr zloc) 2)))))]

   [(fn select [zloc] (and (z/list? zloc)
                           (= 'nth (first (z/sexpr zloc)))
                           (= 'glojure.lang.RT (z/sexpr (z/left zloc)))
                           ))
    (fn visit [zloc] (z/replace zloc `(~'Nth ~@(rest (z/sexpr zloc)))))]

   (sexpr-replace
    '(. clojure.lang.LazilyPersistentVector (create (cons a (cons b (cons c (cons d (cons e (cons f args))))))))
    '(glojure.lang.NewLazilyPersistentVector (cons a (cons b (cons c (cons d (cons e (cons f args))))))))

   (sexpr-replace 'clojure.lang.IDrop 'glojure.lang.IDrop)

   (sexpr-replace 'clojure.lang.Compiler 'glojure.lang.Compiler)
   (sexpr-replace '(. clojure.lang.Compiler (eval form)) '(. clojure.lang.Compiler (Eval form)))

   (sexpr-replace '.alterMeta '.AlterMeta)

   (sexpr-replace 'clojure.lang.Ref 'glojure.lang.Ref)
   (sexpr-replace '(new clojure.lang.Ref x) '(glojure.lang.NewRef x))

   (sexpr-replace '(. e (getKey)) '(. e (GetKey)))
   (sexpr-replace '(. e (getValue)) '(. e (GetValue)))

   (sexpr-replace 'clojure.lang.Named 'glojure.lang.Named)

   ])

(defn rewrite-core [zloc]
  (loop [zloc (z/of-node (z/root zloc))]
    ;; (print "tag" (z/tag zloc))
    ;; (println (z/sexpr zloc))
    (if (z/end? zloc)
      (z/root-string zloc)
      ;; if one of the selectors in replacements matches, replace the form
      (let [zloc (reduce (fn [zloc [select visit]]
                           (if (select zloc)
                             (visit zloc)
                             zloc))
                         zloc
                         replacements)]
        (recur (z/next zloc))))))

;;(rewrite-core zloc)
(print (rewrite-core zloc))
