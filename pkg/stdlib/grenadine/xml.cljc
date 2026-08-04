(ns grenadine.xml
  "A strict, portable XML parser for Maven POM files.

  The parser intentionally implements a small XML subset. It accepts the
  constructs used by POMs and rejects declarations that could resolve external
  content. The result has the same element shape as clojure.data.xml:
  {:tag keyword, :attrs map, :content vector}."
  (:require [clojure.string :as str]))

(defn- position
  [source offset]
  (loop [i 0 line 1 column 1]
    (if (= i offset)
      {:offset offset :line line :column column}
      (if (= "\n" (nth source i))
        (recur (inc i) (inc line) 1)
        (recur (inc i) line (inc column))))))

(defn- xml-error
  [source offset message]
  (throw
   (ex-info message
            (merge {:type :grenadine.xml/parse-error}
                   (position source offset)))))

(defn- starts-at?
  [source offset token]
  (let [token (vec (re-seq #"(?s)." token))
        end (+ offset (count token))]
    (and (<= end (count source))
         (= token (subvec source offset end)))))

(defn- source-slice
  [source start end]
  (apply str (subvec source start end)))

(defn- whitespace?
  [c]
  (contains? #{" " "\t" "\n" "\r"} c))

(defn- skip-whitespace
  [source offset]
  (loop [i offset]
    (if (and (< i (count source)) (whitespace? (nth source i)))
      (recur (inc i))
      i)))

(defn- ascii-letter?
  [c]
  (let [n (int (if (string? c) (first c) c))]
    (or (<= 65 n 90) (<= 97 n 122))))

(defn- ascii-digit?
  [c]
  (let [n (int (if (string? c) (first c) c))]
    (<= 48 n 57)))

(defn- name-start?
  [c]
  (or (ascii-letter? c) (= c "_") (= c ":")))

(defn- name-char?
  [c]
  (or (name-start? c)
      (ascii-digit? c)
      (= c "-")
      (= c ".")))

(defn- parse-name
  [source offset]
  (when (or (= offset (count source))
            (not (name-start? (nth source offset))))
    (xml-error source offset "Expected an XML name"))
  (loop [i (inc offset)]
    (if (and (< i (count source)) (name-char? (nth source i)))
      (recur (inc i))
      [(source-slice source offset i) i])))

(defn- local-keyword
  [name]
  (let [colon (str/last-index-of name ":")]
    (keyword (if colon (subs name (inc colon)) name))))

(defn- digit-value
  [c radix]
  (let [n (int (if (string? c) (first c) c))
        value (cond
                (<= 48 n 57) (- n 48)
                (<= 65 n 70) (+ 10 (- n 65))
                (<= 97 n 102) (+ 10 (- n 97))
                :else -1)]
    (if (< value radix) value -1)))

(defn- parse-integer
  [source offset text radix]
  (when (empty? text)
    (xml-error source offset "Empty numeric character reference"))
  (reduce
   (fn [value c]
     (let [digit (digit-value c radix)]
       (when (neg? digit)
         (xml-error source offset "Invalid numeric character reference"))
       (+ (* value radix) digit)))
   0
   text))

(defn- codepoint->string
  [source offset value]
  (when (or (zero? value)
            (> value 1114111)
            (<= 55296 value 57343))
    (xml-error source offset "Invalid XML character reference"))
  (if (<= value 65535)
    (str (char value))
    (let [n (- value 65536)]
      (str (char (+ 55296 (quot n 1024)))
           (char (+ 56320 (mod n 1024)))))))

(def ^:private named-entities
  {"amp" "&"
   "apos" "'"
   "gt" ">"
   "lt" "<"
   "quot" "\""})

(defn- find-char
  [source offset target]
  (loop [i offset]
    (cond
      (= i (count source)) nil
      (= target (nth source i)) i
      :else (recur (inc i)))))

(defn- decode-entity
  [source offset entity]
  (cond
    (str/starts-with? entity "#x")
    (codepoint->string
     source offset (parse-integer source offset (subs entity 2) 16))

    (str/starts-with? entity "#")
    (codepoint->string
     source offset (parse-integer source offset (subs entity 1) 10))

    :else
    (if-let [value (get named-entities entity)]
      value
      (xml-error source offset (str "Unknown XML entity: &" entity ";")))))

(defn- decode-text
  [source start end]
  (loop [i start pieces [] piece-start start]
    (if (= i end)
      (apply str (conj pieces (source-slice source piece-start end)))
      (if (= "&" (nth source i))
        (let [semi (find-char source (inc i) ";")]
          (when (or (nil? semi) (> semi end))
            (xml-error source i "Unterminated XML entity"))
          (recur (inc semi)
                 (conj pieces
                       (source-slice source piece-start i)
                       (decode-entity source i
                                      (source-slice source (inc i) semi)))
                 (inc semi)))
        (recur (inc i) pieces piece-start)))))

(defn- parse-attribute
  [source offset]
  (let [[name after-name] (parse-name source offset)
        equals-at (skip-whitespace source after-name)]
    (when (or (= equals-at (count source))
              (not= "=" (nth source equals-at)))
      (xml-error source equals-at "Expected '=' after XML attribute name"))
    (let [quote-at (skip-whitespace source (inc equals-at))]
      (when (= quote-at (count source))
        (xml-error source quote-at "Expected quoted XML attribute value"))
      (let [quote-char (nth source quote-at)]
        (when (and (not= quote-char "\"") (not= quote-char "'"))
          (xml-error source quote-at "Expected quoted XML attribute value"))
        (if-let [end (find-char source (inc quote-at) quote-char)]
          [(local-keyword name)
           (decode-text source (inc quote-at) end)
           (inc end)
           name]
          (xml-error source quote-at "Unterminated XML attribute value"))))))

(defn- parse-open-tag
  [source offset]
  (let [[name after-name] (parse-name source offset)]
    (loop [i (skip-whitespace source after-name) attrs {}]
      (cond
        (= i (count source))
        (xml-error source i "Unterminated XML start tag")

        (starts-at? source i "/>")
        [{:tag (local-keyword name) :attrs attrs :content []} (+ i 2) true]

        (= ">" (nth source i))
        [{:tag (local-keyword name) :attrs attrs :content []} (inc i) false]

        :else
        (let [[key value next-i raw-name] (parse-attribute source i)
              namespace-declaration?
              (or (= raw-name "xmlns")
                  (str/starts-with? raw-name "xmlns:"))]
          (when (and (not namespace-declaration?) (contains? attrs key))
            (xml-error source i (str "Duplicate XML attribute: " key)))
          (recur (skip-whitespace source next-i)
                 (if namespace-declaration?
                   attrs
                   (assoc attrs key value))))))))

(defn- find-token
  [source offset token]
  (loop [i offset]
    (cond
      (> (+ i (count token)) (count source)) nil
      (starts-at? source i token) i
      :else (recur (inc i)))))

(defn- append-content
  [stack roots value]
  (if (seq stack)
    [(update-in stack [(dec (count stack)) :content] conj value) roots]
    (if (and (string? value) (str/blank? value))
      [stack roots]
      [stack (conj roots value)])))

(defn parse
  "Parse `source` as strict POM-oriented XML.

  Returns one element map. Whitespace outside the document element is ignored."
  [source]
  (let [source (vec (re-seq #"(?s)." source))]
    (loop [i 0 stack [] roots []]
      (if (= i (count source))
        (cond
          (seq stack)
          (xml-error source i
                     (str "Unclosed XML element: " (:tag (peek stack))))

          (not= 1 (count roots))
          (xml-error source i "XML must contain exactly one document element")

          (not (map? (first roots)))
          (xml-error source i "XML document root must be an element")

          :else
          (first roots))
        (if (= "<" (nth source i))
          (cond
            (starts-at? source i "<!--")
            (if-let [end (find-token source (+ i 4) "-->")]
              (recur (+ end 3) stack roots)
              (xml-error source i "Unterminated XML comment"))

            (starts-at? source i "<?")
            (if-let [end (find-token source (+ i 2) "?>")]
              (recur (+ end 2) stack roots)
              (xml-error source i "Unterminated XML processing instruction"))

            (starts-at? source i "<![CDATA[")
            (if-let [end (find-token source (+ i 9) "]]>")]
              (let [text (source-slice source (+ i 9) end)
                    [next-stack next-roots]
                    (append-content stack roots text)]
                (recur (+ end 3) next-stack next-roots))
              (xml-error source i "Unterminated XML CDATA section"))

            (starts-at? source i "</")
            (let [[name after-name] (parse-name source (+ i 2))
                  close (skip-whitespace source after-name)]
              (when (or (= close (count source))
                        (not= ">" (nth source close)))
                (xml-error source close "Malformed XML closing tag"))
              (when (empty? stack)
                (xml-error source i "Closing tag without an open element"))
              (let [node (peek stack)]
                (when (not= (local-keyword name) (:tag node))
                  (xml-error source i
                             (str "Mismatched XML closing tag: " name)))
                (let [[next-stack next-roots]
                      (append-content (pop stack) roots node)]
                  (recur (inc close) next-stack next-roots))))

            (starts-at? source i "<!")
            (xml-error source i
                       "DOCTYPE and other XML declarations are not supported")

            :else
            (let [[node next-i self-closing?]
                  (parse-open-tag source (inc i))]
              (if self-closing?
                (let [[next-stack next-roots]
                      (append-content stack roots node)]
                  (recur next-i next-stack next-roots))
                (recur next-i (conj stack node) roots))))
          (let [end (or (find-char source i "<") (count source))
                text (decode-text source i end)
                [next-stack next-roots] (append-content stack roots text)]
            (recur end next-stack next-roots)))))))
