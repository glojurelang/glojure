; Licensed to the Apache Software Foundation (ASF) under one or more
; contributor license agreements. See the NOTICE file distributed with this
; work for additional information regarding copyright ownership. The ASF
; licenses this file to you under the Apache License, Version 2.0 (the
; "License"); you may not use this file except in compliance with the License.
; You may obtain a copy of the License at
;
;   http://www.apache.org/licenses/LICENSE-2.0
;
; Unless required by applicable law or agreed to in writing, software
; distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
; WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
; License for the specific language governing permissions and limitations
; under the License.
;
; Portable adaptation of Apache Maven ComparableVersion.java and
; VersionRange.java from maven-3.9.16. Grenadine adaptations Copyright 2026
; Ingy döt Net. See Provenance.md for the exact source mapping and changes.

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

(defn- range-error
  [spec message]
  (throw (ex-info (str "Invalid Maven version range " (pr-str spec)
                       ": " message)
                  {:type :grenadine.version/invalid-range
                   :range spec})))

(defn- whitespace?
  [character]
  (contains? #{\space \tab \newline \return} character))

(defn- skip-whitespace
  [value index]
  (loop [index index]
    (if (and (< index (count value))
             (whitespace? (nth value index)))
      (recur (inc index))
      index)))

(defn- closing-index
  [value start]
  (loop [index start]
    (when (< index (count value))
      (let [character (nth value index)]
        (if (or (= character \]) (= character \)))
          index
          (recur (inc index)))))))

(defn- interval
  [spec opener closer content]
  (let [comma (str/index-of content ",")]
    (if (nil? comma)
      (do
        (when-not (and (= opener \[) (= closer \]) (seq (str/trim content)))
          (range-error spec "an exact range must use [version]"))
        (let [version (str/trim content)]
          {:lower version :lower-inclusive? true
           :upper version :upper-inclusive? true}))
      (do
        (when (str/index-of (subs content (inc comma)) ",")
          (range-error spec "an interval must contain one comma"))
        (let [lower (str/trim (subs content 0 comma))
              upper (str/trim (subs content (inc comma)))]
          (when (and (empty? lower) (empty? upper))
            (range-error spec "an interval must have at least one bound"))
          {:lower (when (seq lower) lower)
           :lower-inclusive? (= opener \[)
           :upper (when (seq upper) upper)
           :upper-inclusive? (= closer \])})))))

(defn parse-version-range
  "Parse a Maven version range into one or more intervals."
  [spec]
  (when-not (and (string? spec) (seq (str/trim spec)))
    (range-error spec "the range must be a non-empty string"))
  (let [value (str/trim spec)]
    (loop [index 0 result []]
      (let [index (skip-whitespace value index)]
        (if (= index (count value))
          (if (seq result)
            result
            (range-error spec "no intervals were found"))
          (let [index
                (if (seq result)
                  (do
                    (when-not (= \, (nth value index))
                      (range-error spec "intervals must be comma-separated"))
                    (let [next-index (skip-whitespace value (inc index))]
                      (when (= next-index (count value))
                        (range-error spec "the range ends after a separator"))
                      next-index))
                  index)
                opener (nth value index)]
            (when-not (or (= opener \[) (= opener \())
              (range-error spec "an interval must start with [ or ("))
            (let [end (closing-index value (inc index))]
              (when-not end
                (range-error spec "an interval is missing its closing bracket"))
              (let [closer (nth value end)
                    content (subs value (inc index) end)]
                (recur (inc end)
                       (conj result (interval spec opener closer content)))))))))))

(defn version-range?
  [value]
  (when (string? value)
    (let [value (str/trim value)]
      (and (seq value)
           (let [first-character (nth value 0)]
             (or (= first-character \[) (= first-character \()))))))

(defn- bound-matches?
  [candidate bound inclusive? lower?]
  (if (nil? bound)
    true
    (let [comparison (compare-versions candidate bound)]
      (if lower?
        (if inclusive? (not (neg? comparison)) (pos? comparison))
        (if inclusive? (not (pos? comparison)) (neg? comparison))))))

(defn in-range?
  "Return true when a concrete version satisfies a parsed or textual range."
  [candidate range]
  (let [intervals (if (string? range) (parse-version-range range) range)]
    (boolean
     (some
      (fn [{:keys [lower lower-inclusive? upper upper-inclusive?]}]
        (and (bound-matches? candidate lower lower-inclusive? true)
             (bound-matches? candidate upper upper-inclusive? false)))
      intervals))))

(defn exact-range-version
  "Return the concrete version represented by `[version]`, otherwise nil."
  [range]
  (let [intervals (if (string? range) (parse-version-range range) range)
        item (first intervals)]
    (when (and (= 1 (count intervals))
               (:lower-inclusive? item)
               (:upper-inclusive? item)
               (= (:lower item) (:upper item)))
      (:lower item))))

(defn snapshot-base-version
  "Return the Maven base-version directory for a timestamped snapshot."
  [value]
  (if-let [[_ base]
           (re-matches #"^(.*)-[0-9]{8}[.][0-9]{6}-[0-9]+$" value)]
    (str base "-SNAPSHOT")
    value))

(defn newer?
  [left right]
  (pos? (compare-versions left right)))
