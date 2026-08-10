(ns grenadine.xml-tree
  "Parser-independent operations over data.xml-shaped element trees."
  (:require [clojure.string :as str]))

(defn parse
  "Parse `source` with the `:xml-parser` function in `opts`."
  [source opts]
  (let [parser (:xml-parser opts)]
    (when-not (fn? parser)
      (throw
       (ex-info "XML parsing requires an :xml-parser function"
                {:type :grenadine.xml/missing-parser
                 :option :xml-parser
                 :value parser})))
    (parser source)))

(defn tag?
  "Return true when `node` has `tag` as its namespace-local element name."
  [node tag]
  (and (map? node)
       (some? (:tag node))
       (= (name tag) (name (:tag node)))))

(defn elements
  "Return the direct child elements of `node` with local name `tag`."
  [node tag]
  (filter #(tag? % tag) (:content node)))

(defn element
  "Return the first direct child element with local name `tag`."
  [node tag]
  (first (elements node tag)))

(defn text
  "Return the trimmed direct text content of `node`, or nil when blank."
  [node]
  (when node
    (let [value (str/trim
                 (apply str (filter string? (:content node))))]
      (when (seq value) value))))
