(ns glojure-rewrite-core
  (:require [rewrite-clj.zip :as z]
            [rewrite-clj.parser :as p]
            [clojure.string :as string]))

(def zloc (z/of-string (slurp (first *command-line-args*))))

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

(defn node-replace
  "Given string representations of old and new, return a replacer function"
  [old new]
  (let [old (p/parse-string old)
        new (p/parse-string new)]
    [(fn select [zloc]
       (= (str (z/node zloc)) (str old)))
     (fn visit [zloc] (z/replace zloc new))]))

(defn sexpr-splice-replace 
  "Replace a single form with multiple forms from a list"
  [old replacement-list]
  [(fn select [zloc] (and (z/sexpr-able? zloc) (= old (z/sexpr zloc))))
   (fn visit [zloc]
     ;; We need to replace the current node with multiple nodes
     ;; First insert all the new nodes after the current one, then remove the current
     (let [parent (z/up zloc)]
       (if (and parent (z/list? parent))
         ;; We're in a list context, can splice
         (loop [zloc zloc
                forms (reverse replacement-list)]
           (if (empty? forms)
             (z/remove zloc)
             (recur (z/insert-right zloc (first forms))
                    (rest forms))))
         ;; Not in a list context, fall back to regular replacement with a list
         (z/replace zloc (cons 'do replacement-list)))))])

(defn sexpr-remove [old]
  [(fn select [zloc] (and (z/sexpr-able? zloc) (= old (z/sexpr zloc))))
   (fn visit [zloc] (z/remove zloc))])

(defn sexpr-replace-any
  [coll new]
  [(fn select [zloc] (and (z/sexpr-able? zloc) (reduce #(or %1 (= %2 (z/sexpr zloc))) false coll)))
   (fn visit [zloc] (z/replace zloc new))])

(defn replace-num-array
  [typ]
  (let [fn-sym (symbol (str typ "_array"))
        new-sym (symbol (str typ "Array"))
        new-sym2 (symbol (str typ "ArrayInit"))]
    [(fn select [zloc]
       (and (z/sexpr-able? zloc)
            (let [sexpr (z/sexpr zloc)]
              (and (list? sexpr)
                   (= (first sexpr) '.)
                   (= (second sexpr) 'clojure.lang.Numbers)
                   (= (nth sexpr 2) fn-sym)))))
     (fn visit [zloc]
       (let [sexpr (z/sexpr zloc)]
         (if (= (count sexpr) 4)
           (z/replace zloc (list '. 'clojure.lang.Numbers new-sym (nth sexpr 3)))
           (z/replace zloc (list '. 'clojure.lang.Numbers new-sym2 (nth sexpr 3) (nth sexpr 4))))))]))

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

(defn omit-symbols [syms]
  [(fn select [zloc] (and (z/list? zloc)
                          (let [sexp (z/sexpr zloc)]
                            (contains? #{'defn 'defn- 'defmacro 'defmacro- 'defprotocol 'extend-protocol 'defmethod}
                                       (first sexp))
                            (contains? syms (second sexp)))))
   (fn visit [zloc] (z/replace zloc '(do)))])

(defn omitp [pred]
  [(fn select [zloc] (pred zloc))
   (fn visit [zloc] (z/remove zloc))])

(defn omit-form [form]
  (omitp #(and (z/sexpr-able? %)
               (= form (z/sexpr %)))))

(defn omit-forms [forms]
  (omitp #(and (z/sexpr-able? %)
               (contains? forms (z/sexpr %)))))

;; Data structures for simple 1:1 replacements
(def namespace-mappings
  {'clojure.core 'glojure.core
   'clojure.string 'glojure.string})

(def type-mappings
  {;; Simple type replacements when appearing alone
   'String 'go/string
   'Long 'go/int64
   'java.lang.Long 'go/int64
   'java.lang.Double 'go/float64
   'Double 'go/float64
   'Float 'go/float32
   'Boolean 'go/bool
   'Character 'github.com$glojurelang$glojure$pkg$lang.Char
   'java.lang.Character 'github.com$glojurelang$glojure$pkg$lang.Char
   'Throwable 'github.com$glojurelang$glojure$pkg$lang.Throwable
   'Object 'github.com$glojurelang$glojure$pkg$lang.Object
   'BigInteger 'math$big.*Int
   'BigDecimal 'github.com$glojurelang$glojure$pkg$lang.*BigDecimal
   'CharSequence 'go/string
   'Class 'reflect.Type
   'Pattern '*Regexp})

(def static-field-mappings
  {'Integer/MIN_VALUE 'math.MinInt
   'Integer/MAX_VALUE 'math.MaxInt
   'Double/POSITIVE_INFINITY '(math.Inf 1)
   'Double/NEGATIVE_INFINITY '(math.Inf -1)
   'Float/POSITIVE_INFINITY '(go/float32 (math.Inf 1))
   'Float/NEGATIVE_INFINITY '(go/float32 (math.Inf -1))})

(defn create-simple-replacements
  "Create sexpr-replace calls from a mapping"
  [mappings]
  (map (fn [[old new]] (sexpr-replace old new)) mappings))

(defn clojure-lang->glojure-pkg
  "Create replacement for clojure.lang.ClassName to glojure package equivalent"
  [class-name & {:keys [pointer? package] 
                 :or {pointer? false 
                      package "github.com$glojurelang$glojure$pkg$lang"}}]
  (sexpr-replace 
    (symbol (str "clojure.lang." class-name))
    (symbol (str package "." (when pointer? "*") class-name))))

(defn clojure-lang-list->glojure-pkg
  "Create replacements for a list of clojure.lang classes"
  [classes & opts]
  (map #(apply clojure-lang->glojure-pkg % opts) classes))

(def replacements
  (concat
    ;; Simple mappings from data structures
    (create-simple-replacements namespace-mappings)
    (create-simple-replacements type-mappings)
    (create-simple-replacements static-field-mappings)
    
    ;; Pattern-based clojure.lang replacements
    (clojure-lang-list->glojure-pkg
      ["IPersistentCollection" "IPersistentList" "IRecord" 
       "NewSymbol" "IReduce" "IPending" "Volatile" "IAtom" "IMapEntry"
       "IPersistentMap" "IPersistentVector" "IPersistentSet" "IMeta"
       "IReduceInit" "IObj" "Keyword" "ISeq" "IEditableCollection"
       "Named" "Counted" "Sequential" "IChunkedSeq"
       "IDrop" "IDeref" "IBlockingDeref"]
      :pointer? false)
    
    (clojure-lang-list->glojure-pkg
      ["Symbol" "Ratio" "MultiFn" "PersistentHashMap" "PersistentHashSet"
       "PersistentVector" "LazySeq" "Var" "Namespace" "Ref" "Agent"
       "BigInt" "BigDecimal"]
      :pointer? true)
    
    [(clojure-lang->glojure-pkg "Fn" :pointer? true :package "github.com$glojurelang$glojure$pkg$runtime")]
    
    ;; All other replacements remain as-is
    [
     ;; ===== Special Clojure.lang Replacements =====
     ;; These don't follow the standard pattern
   (sexpr-replace '(. clojure.lang.PersistentList creator) 'github.com$glojurelang$glojure$pkg$lang.NewList)
   (sexpr-replace '(setMacro) '(SetMacro))


   ;; ===== Math Functions =====
   (sexpr-replace '.isNaN 'math.IsNaN)
   (sexpr-replace 'Double/isNaN 'math.IsNaN)

   ;; ===== Range Constructors =====
   (sexpr-replace '(clojure.lang.LongRange/create end)
                  '(github.com$glojurelang$glojure$pkg$lang.NewLongRange 0 end 1))
   (sexpr-replace '(clojure.lang.LongRange/create start end)
                  '(github.com$glojurelang$glojure$pkg$lang.NewLongRange start end 1))
   (sexpr-replace '(clojure.lang.LongRange/create start end step)
                  '(github.com$glojurelang$glojure$pkg$lang.NewLongRange start end step))

   (sexpr-replace '(clojure.lang.Range/create end)
                  '(github.com$glojurelang$glojure$pkg$lang.NewRange 0 end 1))
   (sexpr-replace '(clojure.lang.Range/create start end)
                  '(github.com$glojurelang$glojure$pkg$lang.NewRange start end 1))
   (sexpr-replace '(clojure.lang.Range/create start end step)
                  '(github.com$glojurelang$glojure$pkg$lang.NewRange start end step))


   ;; ===== Collection Constructors =====
   (sexpr-replace '(. clojure.lang.PersistentHashMap (create keyvals))
                  '(github.com$glojurelang$glojure$pkg$lang.CreatePersistentHashMap keyvals))

   (sexpr-replace '(java.util.ArrayList. n) '(github.com$glojurelang$glojure$pkg$lang.NewArrayList (to-array n)))
   (sexpr-replace '(java.util.ArrayList. coll) '(github.com$glojurelang$glojure$pkg$lang.NewArrayList (to-array coll)))
   (sexpr-replace '(java.util.ArrayList.) '(new github.com$glojurelang$glojure$pkg$lang.ArrayList))

   ;; ===== Other Constructors =====

   (sexpr-replace '(Object.)
                  '(-> nil reflect.StructOf reflect.New .Interface))

   ;; ===== Java Type Mappings =====
   ;; map a bunch of java types to go equivalent
   ;; TODO: once everything passes, see if we can replace with a blanket
   ;; replacement of the clojure.lang prefix.

   (sexpr-replace 'java.util.regex.Matcher
                  'github.com$glojurelang$glojure$pkg$lang.*RegexpMatcher)
   (sexpr-replace 'java.io.PrintWriter
                  'github.com$glojurelang$glojure$pkg$lang.PrintWriter)

   ;; omit PrintWriter-on (defn PrintWriter-on ...)
   (omitp #(and (z/list? %)
                (= 'defn (first (z/sexpr %)))
                (= 'PrintWriter-on (second (z/sexpr %)))))

   (sexpr-replace 'clojure.lang.IReduce
                  'github.com$glojurelang$glojure$pkg$lang.IReduce)
   (sexpr-replace 'clojure.lang.IPending
                  'github.com$glojurelang$glojure$pkg$lang.IPending)
   (sexpr-replace 'clojure.lang.MultiFn
                  'github.com$glojurelang$glojure$pkg$lang.*MultiFn)
   (sexpr-replace 'clojure.lang.Volatile
                  'github.com$glojurelang$glojure$pkg$lang.Volatile)
   (sexpr-replace 'clojure.lang.Volatile.
                  'github.com$glojurelang$glojure$pkg$lang.NewVolatile)
   (sexpr-replace 'clojure.lang.IAtom
                  'github.com$glojurelang$glojure$pkg$lang.IAtom)
   (sexpr-replace 'clojure.lang.IMapEntry
                  'github.com$glojurelang$glojure$pkg$lang.IMapEntry)

   (sexpr-replace 'clojure.lang.PersistentHashMap
                  'github.com$glojurelang$glojure$pkg$lang.*PersistentHashMap)
   (sexpr-replace 'clojure.lang.PersistentHashSet
                  'github.com$glojurelang$glojure$pkg$lang.*PersistentHashSet)
   (sexpr-replace 'clojure.lang.PersistentVector
                  'github.com$glojurelang$glojure$pkg$lang.*PersistentVector)
   (sexpr-replace 'clojure.lang.LazySeq
                  'github.com$glojurelang$glojure$pkg$lang.*LazySeq)

   (sexpr-replace '(clojure.lang.PersistentTreeMap/create keyvals)
                  '(github.com$glojurelang$glojure$pkg$lang.CreatePersistentTreeMap keyvals))

   (sexpr-replace '(clojure.lang.PersistentTreeSet/create keys)
                  '(github.com$glojurelang$glojure$pkg$lang.CreatePersistentTreeSet keys))
   (sexpr-replace '(clojure.lang.PersistentTreeSet/create comparator keys)
                  '(github.com$glojurelang$glojure$pkg$lang.CreatePersistentTreeSetWithComparator comparator keys))

   (sexpr-replace 'clojure.lang.Cycle/create 'github.com$glojurelang$glojure$pkg$lang.NewCycle)

   (sexpr-replace 'clojure.lang.PersistentArrayMap/createAsIfByAssoc
                  'github.com$glojurelang$glojure$pkg$lang.NewPersistentArrayMapAsIfByAssoc)


   (sexpr-replace 'java.util.Map$Entry 'github.com$glojurelang$glojure$pkg$lang.MapEntry)

   (sexpr-replace 'java.net.URI 'net$url.URL)

   (sexpr-replace 'java.util.UUID 'github.com$google$uuid.UUID)

   (sexpr-replace '(java.util.UUID/fromString s)
                  '(let [[uuid err] (github.com$google$uuid.Parse s)]
                     (if err
                       (throw (github.com$glojurelang$glojure$pkg$lang.NewIllegalArgumentError (str "Error parsing UUID: " err)))
                       uuid)))

   (sexpr-replace '(java.util.UUID/randomUUID)
                  '(github.com$google$uuid.NewV7))

   ;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;
   ;; struct map
   (sexpr-replace '(. clojure.lang.PersistentStructMap (createSlotMap keys))
                  '(github.com$glojurelang$glojure$pkg$lang.CreatePersistentStructMapSlotMap keys))
   (sexpr-replace '(. clojure.lang.PersistentStructMap (create s inits))
                  '(github.com$glojurelang$glojure$pkg$lang.CreatePersistentStructMap s inits))
   (sexpr-replace '(. clojure.lang.PersistentStructMap (construct s vals))
                  '(github.com$glojurelang$glojure$pkg$lang.ConstructPersistentStructMap s vals))
   (sexpr-replace '(. clojure.lang.PersistentStructMap (getAccessor s key))
                  '(github.com$glojurelang$glojure$pkg$lang.GetPersistentStructMapAccessor s key))
   ;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;

   (sexpr-replace '(.. (name lib)
                       (replace \- \_)
                       (replace \. \/))
                  '(->
                    (name lib)
                    (strings.ReplaceAll "-" "_")
                    (strings.ReplaceAll "." "/")))
   (sexpr-replace '.startsWith 'strings.HasPrefix)
   (sexpr-replace '(.replace (str ns) \- \_) '(strings.ReplaceAll (str ns) "-" "_"))

   (sexpr-replace '(clojure.lang.Compiler/munge (str s)) '(. clojure.lang.RT (Munge (str s))))

   ;; instance? replacements
   (sexpr-replace "Evaluates x and tests if it is an instance of the class\n    c. Returns true or false"
                  "Evaluates x and tests if it is an instance of the type\n    t. Returns true or false")
   (sexpr-replace '(fn instance? [^Class c x] (. c (isInstance x)))
                  '(fn instance? [t x] (github.com$glojurelang$glojure$pkg$lang.HasType t x)))


   ;; ===== Exception Handling =====
   (sexpr-replace 'Exception. 'github.com$glojurelang$glojure$pkg$lang.NewError)

   (sexpr-replace 'java.lang.UnsupportedOperationException. 'github.com$glojurelang$glojure$pkg$lang.NewUnsupportedOperationError)

   (sexpr-replace 'IllegalArgumentException. 'github.com$glojurelang$glojure$pkg$lang.NewIllegalArgumentError)
   (sexpr-replace 'IllegalArgumentException 'github.com$glojurelang$glojure$pkg$lang.*IllegalArgumentError)
   ;; new Exception
   [(fn select [zloc] (and (z/list? zloc)
                           (let [expr (z/sexpr zloc)]
                             (and (= 'new (first expr))
                                  (= 'Exception (second expr))))))
    (fn visit [zloc]
      (z/replace zloc (concat '(errors.New) (rest (rest (z/sexpr zloc))))))]
   ;; catch Exception
   [(fn select [zloc] (and (z/sexpr-able? zloc)
                           (= 'Exception (z/sexpr zloc))
                           (= 'catch (-> zloc z/left z/sexpr))))
    (fn visit [zloc]
      (z/replace zloc 'go/any))]

   ;; Replace try/catch blocks for Long/valueOf
   [(fn select [zloc] 
      (and (z/list? zloc)
           (= 'try (first (z/sexpr zloc)))
           (let [body (second (z/sexpr zloc))]
             (and (list? body)
                  (= 'Long/valueOf (first body))))))
    (fn visit [zloc]
      (let [s (second (second (z/sexpr zloc)))] ; extract the 's' variable
        (z/replace zloc 
                   `(let [result# (strconv.ParseInt ~s 10 64)]
                      (if (result# 1) nil (result# 0))))))]

   ;; Replace try/catch blocks for Double/valueOf  
   [(fn select [zloc]
      (and (z/list? zloc)
           (= 'try (first (z/sexpr zloc)))
           (let [body (second (z/sexpr zloc))]
             (and (list? body)
                  (= 'Double/valueOf (first body))))))
    (fn visit [zloc]
      (let [s (second (second (z/sexpr zloc)))] ; extract the 's' variable
        (z/replace zloc
                   `(let [result# (strconv.ParseFloat ~s 64)]
                      (if (result# 1) nil (result# 0))))))]

   ;; ===== Metadata Operations =====
   ;; replace .withMeta
   [(fn select [zloc] (and (z/list? zloc) (= '.withMeta (first (z/sexpr zloc)))))
    (fn visit [zloc] (z/replace zloc
                                `(let* [~'res (github.com$glojurelang$glojure$pkg$lang.WithMeta ~@(rest (z/sexpr zloc)))]
                                   (if (~'res 1)
                                     (throw (~'res 1))
                                     (~'res 0)))))]

   ;; ===== RT Function Replacements =====
   (RT-replace 'cons #(cons 'github.com$glojurelang$glojure$pkg$lang.NewCons %))
   (RT-replace 'first #(cons 'github.com$glojurelang$glojure$pkg$lang.First %))
   (RT-replace 'next #(cons 'github.com$glojurelang$glojure$pkg$lang.Next %))
   (RT-replace 'more #(cons 'github.com$glojurelang$glojure$pkg$lang.Rest %))

   [(fn select [zloc] (try
                        (and (symbol? (z/sexpr zloc))
                             (or
                              (and (z/leftmost? zloc) (= 'github.com$glojurelang$glojure$pkg$runtime.RT (-> zloc z/up z/left z/sexpr)))
                              (= 'github.com$glojurelang$glojure$pkg$runtime.RT (-> zloc z/left z/sexpr))))
                        (catch Exception e false)))
    (fn visit [zloc] (z/replace zloc
                                (let [sym (-> zloc z/sexpr str)]
                                  (symbol (str (string/upper-case (first sym)) (subs sym 1))))))]

   (sexpr-replace '.meta '.Meta)
   (sexpr-replace 'clojure.lang.IPersistentMap
                  'github.com$glojurelang$glojure$pkg$lang.IPersistentMap)
   (sexpr-replace 'clojure.lang.IPersistentVector
                  'github.com$glojurelang$glojure$pkg$lang.IPersistentVector)
   (sexpr-replace 'clojure.lang.IPersistentSet
                  'github.com$glojurelang$glojure$pkg$lang.IPersistentSet)
   (sexpr-replace 'clojure.lang.IMeta
                  'github.com$glojurelang$glojure$pkg$lang.IMeta)
   (sexpr-replace 'clojure.lang.IReduceInit
                  'github.com$glojurelang$glojure$pkg$lang.IReduceInit)
   (sexpr-replace 'clojure.lang.IObj
                  'github.com$glojurelang$glojure$pkg$lang.IObj)

   (sexpr-replace 'clojure.lang.Reduced. 'github.com$glojurelang$glojure$pkg$lang.NewReduced)
   (sexpr-replace 'clojure.lang.RT/isReduced 'github.com$glojurelang$glojure$pkg$lang.IsReduced)

   (sexpr-replace '.assoc '.Assoc)


   (sexpr-replace '(. Math (random)) '(math$rand.Float64))

   (sexpr-replace '(. clojure.lang.Var (find sym))
                  '(. github.com$glojurelang$glojure$pkg$runtime.RT (FindVar sym)))

   (sexpr-replace '(. x (get)) '(. x (Get)))
    (sexpr-replace '(. x (set val)) '(. x (Set val)))

   ;; ===== Omissions and Deferrals =====
   (omitp #(and (z/list? %)
                (= 'deftype (first (z/sexpr %)))))
   (omitp #(and (z/list? %)
                (= 'defmethod (first (z/sexpr %)))
                (= 'Eduction (nth (z/sexpr %) 2))))

   ;; Omit eduction function
   (omitp #(and (z/list? %)
                (= 'defn (first (z/sexpr %)))
                (= 'eduction (second (z/sexpr %)))))

   ;; omit default-data-readers for now
   (omitp #(and (z/list? %)
                (= 'def (first (z/sexpr %)))
                (= 'default-data-readers (second (z/sexpr %)))))

   ;; omit Java annotation-related functions (not needed for Go)
   (omit-symbols '#{is-annotation? is-runtime-annotation? descriptor
                     add-annotation process-annotation add-annotations})

   ;; omit tap functions
   (omitp #(and (z/list? %)
                (= 'defonce (first (z/sexpr %)))
                (= 'tap-loop (second (z/sexpr %)))))
   (omitp #(and (z/list? %)
                (= 'defonce (first (z/sexpr %)))
                (= 'tapq (second (z/sexpr %)))))
   (omitp #(and (z/list? %)
                (= 'defn (first (z/sexpr %)))
                (= 'add-tap (second (z/sexpr %)))))
   (omitp #(and (z/list? %)
                (= 'defn (first (z/sexpr %)))
                (= 'tap> (second (z/sexpr %)))))

   [(fn select [zloc] (and (z/list? zloc)
                           (= 'defn- (first (z/sexpr zloc)))
                           (= 'data-reader-urls (second (z/sexpr zloc)))))
    (fn visit [zloc] (z/replace zloc '(defn- data-reader-urls [] ())))]

   (sexpr-replace '(new clojure.lang.Atom x) '(github.com$glojurelang$glojure$pkg$lang.NewAtom x))
   (omitp #(and (z/list? %)
                (let [sexpr (z/sexpr %)]
                  (and (vector? (first sexpr))
                       (= 'atom (first (first sexpr)))
                       (> (count (first sexpr)) 2)))))
   (sexpr-replace '([^clojure.lang.IAtom atom f] (.swap atom f))
                  '([atom f & args] (.swap atom f args)))
   (sexpr-replace '(^github.com$glojurelang$glojure$pkg$lang.IPersistentVector [^github.com$glojurelang$glojure$pkg$lang.IAtom2 atom f] (.swapVals atom f))
                  '([atom f & args] (.swapVals atom f args)))

   ;; ===== Agents =====
   (sexpr-replace '(. clojure.lang.Agent shutdown) '(github.com$glojurelang$glojure$pkg$lang.ShutdownAgents))
   (sexpr-replace 'clojure.lang.Agent 'github.com$glojurelang$glojure$pkg$lang.*Agent)

   ;; ===== Hashing Functions =====
   ;; TODO: these should likely be different
   (sexpr-replace 'clojure.lang.Util/hash 'github.com$glojurelang$glojure$pkg$lang.Hash)
   (sexpr-replace '(. clojure.lang.Util (hasheq x))
                  '(github.com$glojurelang$glojure$pkg$lang.HashEq x))

   (sexpr-replace 'System/identityHashCode 'github.com$glojurelang$glojure$pkg$lang.IdentityHash)

   (sexpr-replace '(String/format fmt (to-array args))
                  '(apply fmt.Sprintf fmt args))

   (sexpr-replace '(clojure.lang.Reflector/prepRet (.getComponentType (class array)) (. Array (get array idx)))
                  '(github.com$glojurelang$glojure$pkg$lang.Get array idx))

   (sexpr-replace '(. Array (set array idx val)) '(github.com$glojurelang$glojure$pkg$lang.SliceSet array idx val))

   ;; ===== Array Operations =====
   [(fn select [zloc] (and (z/sexpr-able? zloc) (= '.reduce (z/sexpr zloc))))
    (fn visit [zloc] (z/replace zloc
                                (let [lst (z/sexpr (z/up zloc))]
                                  (if (= 3 (count lst))
                                    '.Reduce
                                    '.ReduceInit))))]

   (sexpr-replace 'clojure.lang.BigInt/valueOf
                  'github.com$glojurelang$glojure$pkg$lang.NewBigIntFromInt64)
   (sexpr-replace '(BigInteger/valueOf (long x))
                  '(math$big.NewInt (long x)))
   (sexpr-replace '(BigInteger. x) '(github.com$glojurelang$glojure$pkg$lang.NewBigInt x))
   (sexpr-replace '(BigDecimal/valueOf (long x))
                  '(github.com$glojurelang$glojure$pkg$lang.NewBigDecimalFromInt64 (long x)))
   (sexpr-replace '(. BigDecimal valueOf (double x))
                  '(github.com$glojurelang$glojure$pkg$lang.NewBigDecimalFromFloat64 (double x)))
   (sexpr-replace '(BigDecimal. x)
                  '(github.com$glojurelang$glojure$pkg$lang.NewBigDecimal x))
   (sexpr-replace '(BigDecimal. (.numerator ^github.com$glojurelang$glojure$pkg$lang.*Ratio x))
                  '(github.com$glojurelang$glojure$pkg$lang.NewBigDecimalFromRatio x))
   (sexpr-replace 'clojure.lang.BigInt/fromBigInteger
                  'github.com$glojurelang$glojure$pkg$lang.NewBigIntFromGoBigInt)

   (sexpr-replace '.equals '.Equals)

   (sexpr-replace '(clojure.lang.RT/load (.substring path 1))
                  '(. github.com$glojurelang$glojure$pkg$runtime.RT (Load (strings.TrimPrefix path "/"))))
   (sexpr-replace '(. s (substring start)) '(go/slice s start))
   (sexpr-replace '(. s (substring start end)) '(go/slice s start end))

   (sexpr-replace 'clojure.lang.RT/readString 'github.com$glojurelang$glojure$pkg$runtime.RTReadString)

   (sexpr-replace '.lastIndexOf 'strings.LastIndex)

   (sexpr-replace 'clojure.lang.RT/conj 'github.com$glojurelang$glojure$pkg$lang.Conj)
   (sexpr-replace 'withMeta 'WithMeta)

   (sexpr-replace '.asTransient '.AsTransient)
   (sexpr-replace '.persistent '.Persistent)
   (sexpr-replace '.conj '.Conj)

   ;; no need for a special name, as go doesn't have a
   ;; builtin "Equals"
   (sexpr-replace 'clojure.lang.Util/equiv 'github.com$glojurelang$glojure$pkg$lang.Equiv)
   (sexpr-replace 'clojure.lang.Util/equals 'github.com$glojurelang$glojure$pkg$lang.Equals)
   (sexpr-replace '(. clojure.lang.Util (compare x y)) '(github.com$glojurelang$glojure$pkg$lang.Compare x y))

   (sexpr-replace '(. x (meta)) '(.Meta x))

   (sexpr-replace 'clojure.lang.Symbol/intern 'github.com$glojurelang$glojure$pkg$lang.NewSymbol)
   (sexpr-replace '(clojure.lang.Symbol/intern ns name) '(github.com$glojurelang$glojure$pkg$lang.InternSymbol ns name))

   (sexpr-replace '(cond (keyword? name) name
                (symbol? name) (clojure.lang.Keyword/intern ^clojure.lang.Symbol name)
                (string? name) (clojure.lang.Keyword/intern ^String name))
                  '(cond (keyword? name) name
                (symbol? name) (github.com$glojurelang$glojure$pkg$lang.InternKeywordSymbol ^clojure.lang.Symbol name)
                (string? name) (github.com$glojurelang$glojure$pkg$lang.InternKeywordString ^String name)))

   (sexpr-replace '(clojure.lang.Keyword/intern ns name) '(github.com$glojurelang$glojure$pkg$lang.InternKeyword ns name))

   (sexpr-replace '(clojure.lang.Util/identical x nil) '(github.com$glojurelang$glojure$pkg$lang.IsNil x))

   (sexpr-replace '.get '.Get)
   (sexpr-replace '.getName '.Name)
   (sexpr-replace '.concat 'github.com$glojurelang$glojure$pkg$lang.ConcatStrings)
   (sexpr-replace 'clojure.lang.RT/assoc 'github.com$glojurelang$glojure$pkg$lang.Assoc)
   (sexpr-replace 'clojure.lang.RT/subvec 'github.com$glojurelang$glojure$pkg$lang.Subvec)
   (sexpr-replace 'clojure.lang.Util/identical 'github.com$glojurelang$glojure$pkg$lang.Identical)

   (sexpr-replace '(:import (java.util.regex Pattern Matcher) clojure.lang.LazilyPersistentVector)
                  '(:import (regexp *Regexp)))
   (sexpr-replace 'clojure.lang.LazilyPersistentVector/create
                  'github.com$glojurelang$glojure$pkg$lang.CreateLazilyPersistentVector)
   (sexpr-replace 'clojure.lang.LazilyPersistentVector/createOwning
                  'github.com$glojurelang$glojure$pkg$lang.CreateOwningLazilyPersistentVector)
   (sexpr-replace 'LazilyPersistentVector/createOwning
                  'github.com$glojurelang$glojure$pkg$lang.CreateOwningLazilyPersistentVector)

   (sexpr-replace '(. clojure.lang.RT (seq coll)) '(github.com$glojurelang$glojure$pkg$lang.Seq coll))
   (sexpr-replace '(list 'new 'clojure.lang.LazySeq (list* '^{:once true} fn* [] body))
                  '(list 'github.com$glojurelang$glojure$pkg$lang.NewLazySeq (list* '^{:once true} fn* [] body)))
   (sexpr-replace 'clojure.lang.RT/count 'github.com$glojurelang$glojure$pkg$lang.Count)

   (sexpr-replace 'clojure.lang.IChunkedSeq 'github.com$glojurelang$glojure$pkg$lang.IChunkedSeq)
   (sexpr-replace 'clojure.lang.ChunkBuffer.
                  'github.com$glojurelang$glojure$pkg$lang.NewChunkBuffer)
   (sexpr-replace 'clojure.lang.ChunkedCons.
                  'github.com$glojurelang$glojure$pkg$lang.NewChunkedCons)

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
                                  `(github.com$glojurelang$glojure$pkg$lang.Apply
                                    ~(nth sexpr 1)
                                    ~(nth (nth sexpr 2) 1)))))]

   (sexpr-replace '(. clojure.lang.RT (get map key)) '(github.com$glojurelang$glojure$pkg$lang.Get map key))
   (sexpr-replace '(. clojure.lang.RT (get map key not-found)) '(github.com$glojurelang$glojure$pkg$lang.GetDefault map key not-found))

   ;; TODO: replace these using the RT-replace function!
   (sexpr-replace '(. clojure.lang.RT (keys map)) '(github.com$glojurelang$glojure$pkg$lang.Keys map))
   (sexpr-replace '(. clojure.lang.RT (vals map)) '(github.com$glojurelang$glojure$pkg$lang.Vals map))
   (sexpr-replace '(. clojure.lang.RT (seq map)) '(github.com$glojurelang$glojure$pkg$lang.Seq map))

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
   (sexpr-replace '(. x (toString)) '(github.com$glojurelang$glojure$pkg$lang.ToString x))
   (sexpr-replace '.toString 'github.com$glojurelang$glojure$pkg$lang.ToString)
   (sexpr-replace 'getName 'Name)
   (sexpr-replace 'getNamespace 'Namespace)
   (sexpr-replace '.hasRoot '.HasRoot)
   (sexpr-replace '.resetMeta '.ResetMeta)


   ;; ===== Multi-methods =====
   [(fn select [zloc] (and (z/list? zloc)
                           (let [sexpr (z/sexpr zloc)]
                             (and
                              (= 'new (first sexpr))
                              (= 'clojure.lang.MultiFn (second sexpr))))))
    (fn visit [zloc] (-> zloc
                         z/down
                         (z/replace 'github.com$glojurelang$glojure$pkg$lang.NewMultiFn)
                         z/right
                         z/remove))]
   (sexpr-replace 'clojure.lang.MultiFn 'github.com$glojurelang$glojure$pkg$lang.*MultiFn)
   (sexpr-replace 'addMethod 'AddMethod)
   (sexpr-replace 'preferMethod 'PreferMethod)

   (let [new-isa "(defn isa?
  \"Returns true if (= child parent), or child is directly or indirectly derived from
  parent, either via a Java type inheritance relationship or a
  relationship established via derive. h must be a hierarchy obtained
  from make-hierarchy, if not supplied defaults to the global
  hierarchy\"
  {:added \"1.0\"}
  ([child parent] (isa? global-hierarchy child parent))
  ([h child parent]
   (or (= child parent)
       (and (class? parent) (class? child)
            (. ^reflect.Type child AssignableTo parent))
       (contains? ((:ancestors h) child) parent)
       (and (class? child) (some #(contains? ((:ancestors h) %) parent) (supers child)))
       (and (vector? parent) (vector? child)
            (= (count parent) (count child))
            (loop [ret true i 0]
              (if (or (not ret) (= i (count parent)))
                ret
                (recur (isa? h (child i) (parent i)) (inc i))))))))
"
         new-node (p/parse-string new-isa)]
     [(fn select [zloc] (and (z/list? zloc)
                             (let [sexpr (z/sexpr zloc)]
                               (and
                                (= 'defn (first sexpr))
                                (= 'isa? (second sexpr))))))
      (fn visit [zloc] (z/replace zloc new-node))])


   (sexpr-replace '(System/getProperty "line.separator") '"\\n")
   (sexpr-replace 'clojure.lang.ISeq 'github.com$glojurelang$glojure$pkg$lang.ISeq)
   (sexpr-replace 'clojure.lang.IEditableCollection 'github.com$glojurelang$glojure$pkg$lang.IEditableCollection)
   (sexpr-replace 'clojure.core/import* 'github.com$glojurelang$glojure$pkg$lang.Import)

   ;; ===== Import Omissions =====
   (omit-forms '#{(import '(java.lang.reflect Array))
                  (import clojure.lang.ExceptionInfo clojure.lang.IExceptionInfo)
                  (import '(java.util.concurrent BlockingQueue LinkedBlockingQueue))
                  (import '(java.io Writer))})

   (sexpr-replace '(. System (nanoTime)) '(.UnixNano (time.Now)))

   (sexpr-replace '(.. Runtime getRuntime availableProcessors)
                  '(runtime.NumCPU))

   (sexpr-replace 'clojure.lang.RT/longCast 'github.com$glojurelang$glojure$pkg$lang.LongCast)
   (sexpr-replace 'clojure.lang.RT/byteCast 'github.com$glojurelang$glojure$pkg$lang.ByteCast)
   (sexpr-replace 'clojure.lang.RT/uncheckedByteCast 'github.com$glojurelang$glojure$pkg$lang.UncheckedByteCast)
   (sexpr-replace 'clojure.lang.RT/shortCast 'github.com$glojurelang$glojure$pkg$lang.ShortCast)
   (sexpr-replace 'clojure.lang.RT/uncheckedShortCast 'github.com$glojurelang$glojure$pkg$lang.UncheckedShortCast)
   (sexpr-replace 'clojure.lang.RT/doubleCast 'github.com$glojurelang$glojure$pkg$lang.AsFloat64) ;; todo: checked version
   (sexpr-replace 'clojure.lang.RT/uncheckedDoubleCast 'github.com$glojurelang$glojure$pkg$lang.AsFloat64)
   (sexpr-replace 'clojure.lang.RT/floatCast 'github.com$glojurelang$glojure$pkg$lang.FloatCast)
   (sexpr-replace 'clojure.lang.RT/uncheckedFloatCast 'github.com$glojurelang$glojure$pkg$lang.UncheckedFloatCast)

   (sexpr-replace "clojure.core" "glojure.core")
   (sexpr-replace 'clojure.core/name 'glojure.core/name)

   ;; ===== Number Type Checks =====
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
                     (github.com$glojurelang$glojure$pkg$lang.IsInteger n))
                  )


   (sexpr-replace '(clojure.lang.RT/booleanCast x) '(. github.com$glojurelang$glojure$pkg$runtime.RT (BooleanCast x)))
   ;; TODO: meet unchecked behavior?
   (sexpr-replace 'clojure.lang.RT/uncheckedLongCast
                  'github.com$glojurelang$glojure$pkg$lang.UncheckedLongCast)
   (sexpr-replace 'clojure.lang.RT/uncheckedIntCast
                  'github.com$glojurelang$glojure$pkg$lang.UncheckedIntCast)

   [(fn select [zloc] (try
                        (and (symbol? (z/sexpr zloc))
                             (or
                              (and (z/leftmost? zloc) (= 'github.com$glojurelang$glojure$pkg$lang.Numbers (-> zloc z/up z/left z/sexpr)))
                              (= 'github.com$glojurelang$glojure$pkg$lang.Numbers (-> zloc z/left z/sexpr))))
                        (catch Exception e false)))
    (fn visit [zloc] (z/replace zloc
                                (let [sym (-> zloc z/sexpr str)]
                                  (symbol (str (first sym) (subs sym 1))))))]
   (sexpr-splice-replace 'clojure.lang.Numbers/gt
                         ['.Gt 'github.com$glojurelang$glojure$pkg$lang.Numbers])

   (sexpr-replace 'clojure.lang.Numbers 'github.com$glojurelang$glojure$pkg$lang.Numbers)
   (sexpr-replace '(cast Number x) '(github.com$glojurelang$glojure$pkg$lang.MustAsNumber x))
   (sexpr-replace '(instance? Number x) '(github.com$glojurelang$glojure$pkg$lang.IsNumber x))

   (sexpr-replace '(. clojure.lang.Numbers (minus x))
                  '(* -1 x)) ;; TODO: unary minus
   (sexpr-replace '(. clojure.lang.Numbers (minusP x))
                  '(*' -1 x))
   (sexpr-replace 'clojure.lang.Numbers/isZero
                  'github.com$glojurelang$glojure$pkg$lang.IsZero)
   (sexpr-replace 'clojure.lang.Numbers/abs
                  'github.com$glojurelang$glojure$pkg$lang.Abs)

   (sexpr-replace 'unchecked_add 'UncheckedAdd)
   (sexpr-replace 'unchecked_dec 'UncheckedDec)
   (sexpr-replace 'unchecked_int_divide 'UncheckedIntDivide)
   (sexpr-replace '(unchecked_minus x) '(Unchecked_negate x))

   ;; ===== Numeric Array Replacements =====
   (replace-num-array 'char)
   (replace-num-array 'byte)
   (replace-num-array 'short)
   (replace-num-array 'int)
   (replace-num-array 'long)
   (replace-num-array 'float)
   (replace-num-array 'double)
   (replace-num-array 'boolean)

   (sexpr-replace 'clojure.core/cond 'glojure.core/cond)

   (sexpr-replace 'clojure.lang.Keyword 'github.com$glojurelang$glojure$pkg$lang.Keyword)

   (sexpr-replace 'clojure.lang.RT 'github.com$glojurelang$glojure$pkg$runtime.RT)
   (sexpr-replace '(nextID) '(NextID))

   (sexpr-replace '(nth coll index not-found) '(NthDefault coll index not-found))

   [(fn select [zloc] (and (z/list? zloc)
                           (= '. (first (z/sexpr zloc)))
                           (= 'clojure.lang.Symbol (second (z/sexpr zloc)))
                           (= 'intern (first (nth (z/sexpr zloc) 2)))
                           ))
    (fn visit [zloc] (z/replace zloc `(github.com$glojurelang$glojure$pkg$lang.NewSymbol ~@(rest (nth (z/sexpr zloc) 2)))))]

   [(fn select [zloc] (and (z/list? zloc)
                           (= 'nth (first (z/sexpr zloc)))
                           (= 'github.com$glojurelang$glojure$pkg$runtime.RT (z/sexpr (z/left zloc)))
                           ))
    (fn visit [zloc] (z/replace zloc `(~'Nth ~@(rest (z/sexpr zloc)))))]

   (sexpr-replace
    '(. clojure.lang.LazilyPersistentVector (create (cons a (cons b (cons c (cons d (cons e (cons f args))))))))
    '(github.com$glojurelang$glojure$pkg$lang.CreateLazilyPersistentVector (cons a (cons b (cons c (cons d (cons e (cons f args))))))))

   (sexpr-replace 'clojure.lang.IDrop 'github.com$glojurelang$glojure$pkg$lang.IDrop)

   (sexpr-replace 'clojure.lang.Compiler 'github.com$glojurelang$glojure$pkg$runtime.Compiler)
   (sexpr-replace '(. clojure.lang.Compiler (eval form)) '(. clojure.lang.Compiler (Eval form)))
   (sexpr-replace '(clojure.lang.Compiler/maybeResolveIn (the-ns ns) sym)
                  '(. github.com$glojurelang$glojure$pkg$runtime.Compiler maybeResolveIn (the-ns ns) sym))
   (sexpr-replace 'clojure.lang.Compiler$CompilerException.
                  'github.com$glojurelang$glojure$pkg$lang.NewCompilerError)

   (sexpr-replace '.alterMeta '.AlterMeta)

   (sexpr-replace 'clojure.lang.Ref 'github.com$glojurelang$glojure$pkg$lang.*Ref)
   (sexpr-replace 'clojure.lang.IDeref 'github.com$glojurelang$glojure$pkg$lang.IDeref)

   (sexpr-replace '(new clojure.lang.Ref x) '(github.com$glojurelang$glojure$pkg$lang.NewRef x))
   (sexpr-replace 'clojure.lang.LockingTransaction 'github.com$glojurelang$glojure$pkg$lang.LockingTransaction)
   (sexpr-replace 'runInTransaction 'RunInTransaction)

   (sexpr-replace '(. e (getKey)) '(. e (GetKey)))
   (sexpr-replace '(. e (getValue)) '(. e (GetValue)))

   (sexpr-replace '.deref '.Deref)
   (sexpr-replace '(. ref (commute fun args)) '(. ref (Commute fun args)))

   (sexpr-replace 'clojure.lang.Named 'github.com$glojurelang$glojure$pkg$lang.Named)

   (sexpr-replace 'clojure.lang.Namespace/find 'github.com$glojurelang$glojure$pkg$lang.FindNamespace)
   (sexpr-replace 'clojure.lang.Namespace/remove
                  'github.com$glojurelang$glojure$pkg$lang.RemoveNamespace)

   (sexpr-replace '(clojure.lang.Repeat/create x) '(github.com$glojurelang$glojure$pkg$lang.NewRepeat x))
   (sexpr-replace '(clojure.lang.Repeat/create n x) '(github.com$glojurelang$glojure$pkg$lang.NewRepeatN n x))

   (sexpr-replace '.charAt 'github.com$glojurelang$glojure$pkg$lang.CharAt)

   ;; ===== File Loading and Module Omissions =====
   ;;; TODO: implement load for embedded files!
   (sexpr-replace '(load "core_proxy") '(do))
   (sexpr-replace '(load "genclass") '(do))
   (sexpr-replace '(load "core/protocols") '(load "protocols"))
   (sexpr-replace '(load "gvec") '(do))
   (sexpr-replace '(load "uuid") '(do))

   (sexpr-replace '(require '[clojure.java.io :as jio])
                  '(require '[glojure.go.io :as gio]))
   (sexpr-replace 'jio/reader 'gio/reader)
   (sexpr-replace 'jio/copy 'gio/copy)
   (sexpr-replace 'jio/writer 'gio/writer)
   (sexpr-replace 'Reader 'io.Reader)

   (sexpr-replace 'java.io.StringWriter 'strings.Builder)
   (sexpr-replace '(java.io.StringWriter.)
                  '(new strings.Builder))

   (sexpr-replace 'java.io.Writer 'io.Writer)

   (omit-symbols
    '#{when-class
       Inst
       clojure.core/Inst
       clojure.core.protocols/IKVReduce
       })

   (sexpr-replace '(when-class "java.sql.Timestamp" (load "instant")) '(do))

   (sexpr-replace '.indexOf 'strings.Index)

   (sexpr-replace 'clojure.lang.Counted 'github.com$glojurelang$glojure$pkg$lang.Counted)

   (sexpr-replace 'clojure.core/in-ns 'glojure.core/in-ns)
   (sexpr-replace 'clojure.core/refer 'glojure.core/refer)

   (sexpr-replace 'clojure.lang.Var 'github.com$glojurelang$glojure$pkg$lang.*Var)
   (sexpr-replace 'clojure.lang.Namespace 'github.com$glojurelang$glojure$pkg$lang.*Namespace)

   (sexpr-replace 'clojure.lang.Sequential 'github.com$glojurelang$glojure$pkg$lang.Sequential)

   (sexpr-replace '(. *ns* (refer (or (rename sym) sym) v))
                  '(. *ns* (Refer (or (rename sym) sym) v)))

   (sexpr-replace '.getMappings '.Mappings)
   (sexpr-replace '.ns '.Namespace)
   (sexpr-replace '.isPublic '.IsPublic)
   (sexpr-replace '.addAlias '.AddAlias)

   [(fn select [zloc] (and (z/sexpr-able? zloc) (= 'pushThreadBindings (z/sexpr zloc))))
    (fn visit [zloc] (z/replace (-> zloc z/up z/up)
                                '(github.com$glojurelang$glojure$pkg$lang.PushThreadBindings {})))]
   (sexpr-replace '(. clojure.lang.Var (popThreadBindings)) '(github.com$glojurelang$glojure$pkg$lang.PopThreadBindings))
   (sexpr-replace 'clojure.lang.Var/popThreadBindings 'github.com$glojurelang$glojure$pkg$lang.PopThreadBindings)
   (sexpr-replace 'clojure.lang.Var/pushThreadBindings 'github.com$glojurelang$glojure$pkg$lang.PushThreadBindings)

   ;; support pmap
   (sexpr-replace 'clojure.lang.Var/cloneThreadBindingFrame
                  'github.com$glojurelang$glojure$pkg$lang.CloneThreadBindingFrame)
   (sexpr-replace 'clojure.lang.Var/resetThreadBindingFrame
                  'github.com$glojurelang$glojure$pkg$lang.ResetThreadBindingFrame)
   [(fn select [zloc] (and (z/list? zloc) (= 'future-call (second (z/sexpr zloc)))))
    (fn visit [zloc] (z/replace zloc
                                '(defn future-call 
                                   "Takes a function of no args and yields a future object that will
  invoke the function in another thread, and will cache the result and
  return it on all subsequent calls to deref/@. If the computation has
  not yet finished, calls to deref/@ will block, unless the variant
  of deref with timeout is used. See also - realized?."
                                   {:added "1.1"
                                    :static true}
                                   [f]
                                   (let [f (binding-conveyor-fn f)
                                         fut (github.com$glojurelang$glojure$pkg$lang.AgentSubmit f)]
                                     fut))))]
   (sexpr-replace 'java.util.concurrent.TimeUnit/MILLISECONDS
                  'time.Millisecond)
   (sexpr-replace 'java.util.concurrent.TimeoutException
                  'github.com$glojurelang$glojure$pkg$lang.TimeoutError)
   (sexpr-replace 'clojure.lang.IBlockingDeref
                  'github.com$glojurelang$glojure$pkg$lang.IBlockingDeref)
   [(fn select [zloc] (and (z/list? zloc)
                           (= '.deref (first (z/sexpr zloc)))
                           (= 4 (count (z/sexpr zloc)))))
    (fn visit [zloc] (z/replace zloc
                                '(.DerefWithTimeout ref timeout-ms timeout-val)))]

   ;; TODO: special tags
   (sexpr-replace '(clojure.lang.Compiler$HostExpr/maybeSpecialTag tag) nil)
   (sexpr-replace '(clojure.lang.Compiler$HostExpr/maybeClass tag false) nil)

   ;; TODO: clojure version
   (omit-symbols '#{clojure-version})
   (omitp #(and (z/list? %) (= '*clojure-version* (second (z/sexpr %)))))
   [(fn select [zloc] (and (z/sexpr-able? zloc) (= 'version-string (z/sexpr zloc))))
    (fn visit [zloc] (z/replace (-> zloc z/up z/up) '(do)))]

   (sexpr-replace '(. x (getClass))
                  '(github.com$glojurelang$glojure$pkg$lang.TypeOf x))

   ;; ===== Core Print Replacements =====


   (sexpr-replace '(.isArray c) false)
   ;; (sexpr-replace '(print-method (.Name c) w) 'TODO)
   ;; (sexpr-replace '(github.com$glojurelang$glojure$pkg$lang.WriteWriter w (.Name c)) 'TODO)

   (sexpr-replace '(prefer-method print-dup java.util.Map clojure.lang.Fn) '(do))
   (sexpr-replace '(prefer-method print-dup java.util.Collection clojure.lang.Fn) '(do))
   (sexpr-replace '(prefer-method print-method clojure.lang.ISeq java.util.Collection) '(do))
   (sexpr-replace '(prefer-method print-dup clojure.lang.ISeq java.util.Collection) '(do))
   (sexpr-replace '(prefer-method print-dup clojure.lang.IPersistentCollection java.util.Collection) '(do))

   (sexpr-replace-any
    '[
      (prefer-method print-method clojure.lang.IPersistentCollection java.util.Collection)
      (prefer-method print-method clojure.lang.IPersistentCollection java.util.RandomAccess)
      (prefer-method print-method java.util.RandomAccess java.util.List)
      (prefer-method print-method clojure.lang.IPersistentCollection java.util.Map)
      (prefer-method print-method clojure.lang.IRecord java.util.Map)
      (prefer-method print-dup clojure.lang.IPersistentCollection java.util.Map)
      (prefer-method print-dup clojure.lang.IRecord java.util.Map)
      ]
    '(do))

   (sexpr-replace 'java.util.regex.Pattern 'regexp.*Regexp)
   (sexpr-replace 'clojure.lang.BigInt 'github.com$glojurelang$glojure$pkg$lang.*BigInt)
   (sexpr-replace 'java.math.BigDecimal 'github.com$glojurelang$glojure$pkg$lang.*BigDecimal)

   (sexpr-replace '.write 'github.com$glojurelang$glojure$pkg$lang.WriteWriter)
   (sexpr-replace '.append 'github.com$glojurelang$glojure$pkg$lang.AppendWriter)
   (sexpr-replace '(. *out* (append \space)) '(github.com$glojurelang$glojure$pkg$lang.AppendWriter *out* \space))
   (sexpr-replace '(. *out* (append system-newline))
                  '(github.com$glojurelang$glojure$pkg$lang.AppendWriter *out* system-newline))
   (sexpr-replace '(. *out* (flush)) '(. *out* (Sync)))

   (omit-symbols '#{primitives-classnames})

   (sexpr-replace '(.getInterfaces c) nil) ;; no such concept in go
   (sexpr-replace '(.getSuperclass c) nil) ;; no such concept in go

   ;; Omit some methods
   [(fn select [zloc] (and (z/list? zloc)
                           (let [sexpr (z/sexpr zloc)]
                             (and (= 'defmethod (first sexpr))
                                  (contains? #{'print-method 'print-dup} (second sexpr))
                                  (contains? #{'java.util.Collection
                                               'java.util.Map
                                               'java.util.List
                                               'java.util.RandomAccess
                                               'java.util.Set
                                               'clojure.lang.LazilyPersistentVector
                                               'Class
                                               'StackTraceElement
                                               'Throwable
                                               ;; TODO: support
                                               'clojure.lang.TaggedLiteral
                                               'clojure.lang.ReaderConditional
                                               } (nth sexpr 2))))))
    (fn visit [zloc] (z/replace zloc '(do)))]

   ;; Implement print-* for number types
   [(fn select [zloc] (and (z/list? zloc)
                           (let [sexpr (z/sexpr zloc)]
                             (and (= 'defmethod (first sexpr))
                                  (contains? #{'print-method 'print-dup} (second sexpr))
                                  (= (nth sexpr 2) 'Number)))))
    (fn visit [zloc]
      (loop [ints '[go/int go/uint go/uint8 go/uint16 go/uint32 go/uint64 go/int8 go/int16 go/int32 go/int64 go/byte go/rune github.com$glojurelang$glojure$pkg$lang.*Ratio]
             zloc zloc]
        (if (empty? ints)
          (z/remove zloc)
          (recur (rest ints)
                 (-> zloc
                     (z/insert-left
                      `(~'defmethod ~'print-method ~(first ints) [~'o, ~'w]
                        (~'.write ~'w (~'str ~'o))))
                     (z/insert-newline-left))
                 ))))]

   ;;; replace all clojure. symbols with glojure.
   [(fn select [zloc] (and (z/sexpr-able? zloc)
                           (let [sexpr (z/sexpr zloc)]
                             (and (symbol? sexpr)
                                  (string/starts-with? (name sexpr) "clojure.")))))
    (fn visit [zloc] (z/replace zloc (-> zloc
                                         z/sexpr
                                         name
                                         (string/replace "clojure." "glojure.")
                                         symbol)))]

   ;; ===== Test.clj Replacements =====

   (sexpr-remove '[clojure.stacktrace :as stack])

   [(fn select [zloc] (and (z/list? zloc)
                           (= 'stacktrace-file-and-line
                              (first (z/sexpr zloc)))))
    (fn visit [zloc] (z/replace zloc '{}))]

   ;; ===== Regular Expression Replacements =====
   (sexpr-replace '(.split re s)
                  '(.split re s -1))

   (sexpr-replace '(.length s) '(count s))
   (sexpr-replace '(.length match) '(count match))

   (sexpr-replace '(. java.util.regex.Pattern (compile s))
                  '(regexp.MustCompile s))
   (sexpr-replace '(. re (matcher s))
                  '(github.com$glojurelang$glojure$pkg$lang.NewRegexpMatcher re s))

   (sexpr-replace '(. m (group c)) '(. m (groupInt c)))

   (node-replace "(.pattern ^java.util.regex.Pattern p)"
                 "(.String ^regexp.*Regexp p)")

   ;; Arrays.sort replacement for Glojure sort function
   (sexpr-replace '(. java.util.Arrays (sort a comp))
                  '(github.com$glojurelang$glojure$pkg$lang.SortSlice a comp))

   ;; comparators are simple functions in Glojure
   (sexpr-replace '(. comp (compare (keyfn x) (keyfn y)))
                  '(comp (keyfn x) (keyfn y)))
   ]))

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
