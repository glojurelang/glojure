(ns grenadine.version
  "Portable Maven-style version ordering."
  (:require [clojure.string :as str]))

(def ^:private qualifier-aliases
  {"a" "alpha"
   "b" "beta"
   "m" "milestone"
   "cr" "rc"
   "ga" ""
   "final" ""
   "release" ""})

(def ^:private qualifier-order
  {"alpha" 0
   "beta" 1
   "milestone" 2
   "rc" 3
   "snapshot" 4
   "" 5
   "sp" 6})

(defn- digit?
  [c]
  (let [n (int c)]
    (<= 48 n 57)))

(defn- normalize-number
  [value]
  (loop [i 0]
    (if (and (< i (dec (count value)))
             (= \0 (nth value i)))
      (recur (inc i))
      (subs value i))))

(defn- token
  [value numeric? separator]
  (if numeric?
    {:kind :number :value (normalize-number value) :separator separator}
    (let [value (str/lower-case value)]
      {:kind :qualifier
       :value (get qualifier-aliases value value)
       :separator separator})))

(defn- tokenize
  [version]
  (loop [i 0
         start 0
         numeric? nil
         current-separator nil
         next-separator nil
         result []]
    (if (= i (count version))
      (let [result
            (if (< start i)
              (conj result
                    (token (subs version start i)
                           numeric?
                           current-separator))
              result)]
        ;; ComparableVersion treats 1-1 as the normalized form 1.0-1.
        (if (and (= 2 (count result))
                 (= :number (:kind (first result)))
                 (= :number (:kind (second result)))
                 (= :hyphen (:separator (second result))))
          [(first result)
           {:kind :number :value "0" :separator :dot}
           (second result)]
          result))
      (let [c (nth version i)
            separator? (or (= c \.) (= c \-))
            current-numeric? (digit? c)]
        (cond
          separator?
          (recur (inc i)
                 (inc i)
                 nil
                 nil
                 (if (= c \-) :hyphen :dot)
                 (if (< start i)
                   (conj result
                         (token (subs version start i)
                                numeric?
                                current-separator))
                   result))

          (nil? numeric?)
          (recur (inc i)
                 start
                 current-numeric?
                 next-separator
                 nil
                 result)

          (= numeric? current-numeric?)
          (recur (inc i)
                 start
                 numeric?
                 current-separator
                 next-separator
                 result)

          :else
          (recur (inc i)
                 i
                 current-numeric?
                 :transition
                 nil
                 (conj result
                       (token (subs version start i)
                              numeric?
                              current-separator))))))))

(defn- zero-token?
  [item]
  (or (and (= :number (:kind item)) (= "0" (:value item)))
      (and (= :qualifier (:kind item)) (= "" (:value item)))))

(defn- normalize-tokens
  [items]
  (loop [items (vec items)]
    (if (and (seq items) (zero-token? (peek items)))
      (recur (pop items))
      items)))

(defn- compare-number
  [left right]
  (let [length-comparison (compare (count left) (count right))]
    (if (zero? length-comparison)
      (compare left right)
      length-comparison)))

(defn- qualifier-key
  [value]
  (if-let [rank (get qualifier-order value)]
    [0 rank ""]
    [1 0 value]))

(defn- compare-qualifier
  [left right]
  (compare (qualifier-key left) (qualifier-key right)))

(defn- compare-item
  [left right]
  (cond
    (nil? left)
    (cond
      (nil? right) 0
      (= :number (:kind right))
      (- (compare-number (:value right) "0"))
      :else
      (- (compare-qualifier (:value right) "")))

    (nil? right)
    (- (compare-item right left))

    (= (:kind left) (:kind right))
    (if (= :number (:kind left))
      (let [number-comparison
            (compare-number (:value left) (:value right))]
        (if (zero? number-comparison)
          (compare
           (if (= :hyphen (:separator left)) 0 1)
           (if (= :hyphen (:separator right)) 0 1))
          number-comparison))
      (compare-qualifier (:value left) (:value right)))

    (= :number (:kind left)) 1
    :else -1))

(defn compare-versions
  "Compare two Maven version strings, returning a negative number, zero, or a
  positive number.

  This covers Maven's numeric segments, digit/letter transitions, standard
  qualifier aliases, qualifier ordering, and arbitrary-length numeric fields."
  [left right]
  (let [left-items (normalize-tokens (tokenize left))
        right-items (normalize-tokens (tokenize right))
        limit (max (count left-items) (count right-items))]
    (loop [i 0]
      (if (= i limit)
        0
        (let [comparison
              (compare-item (get left-items i) (get right-items i))]
          (if (zero? comparison)
            (recur (inc i))
            comparison))))))

(defn newer?
  [left right]
  (pos? (compare-versions left right)))
