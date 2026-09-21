(ns glojure.test-glojure.jvm-interop
  (:use clojure.test))

;; JVM-flavored user code must run unrewritten. Every form here also
;; appears in pkg/runtime/testdata/codegen/test/jvm_interop.clj so the
;; AOT path is checked with the same corpus.

(deftest t-static-methods
  (is (= 3 (Math/abs -3)))
  (is (= true (Boolean/parseBoolean "true")))
  (is (= 42 (Integer/parseInt "42")))
  (is (= 42 (Long/parseLong "42")))
  (is (= 1.5 (Double/parseDouble "1.5")))
  (is (= "7" (String/valueOf 7)))
  (is (Character/isDigit \7)))

(deftest t-fully-qualified-static-methods
  (is (= 3 (java.lang.Math/abs -3)))
  (is (= true (java.lang.Boolean/parseBoolean "true")))
  (is (= 42 (java.lang.Integer/parseInt "42")))
  (is (java.util.regex.Pattern/matches "a+" "aaa"))
  (is (= "a\\.b" (java.util.regex.Pattern/quote "a.b"))))

(deftest t-static-fields
  (is (= 2147483647 Integer/MAX_VALUE))
  (is (= -9223372036854775808 Long/MIN_VALUE))
  (is (= 2147483647 java.lang.Integer/MAX_VALUE))
  (is (Double/isNaN Double/NaN))
  (is (= 2 java.util.regex.Pattern/CASE_INSENSITIVE)))

(deftest t-constructors
  (is (= 42 (Integer. "42")))
  (is (= 42 (Long. "42")))
  (is (= 7 (Long. 7)))
  (is (= 1.5 (Double. "1.5")))
  (is (= \a (Character. \a)))
  (is (= true (Boolean. "true")))
  (is (= "a+" (.pattern (java.util.regex.Pattern. "a+"))))
  (is (= "a+" (.pattern (java.util.regex.Pattern/compile "a+")))))

(deftest t-instance-methods
  (is (= "ABC" (.toUpperCase "abc")))
  (is (= 3 (.length "abc")))
  (is (= "b,c" (.substring "a,b,c" 2)))
  (is (= "x" (.trim "  x  ")))
  (is (= [97 98] (vec (.getBytes "ab")))))

(deftest t-classes
  (is (instance? Number 1))
  (is (instance? Number 1.5))
  (is (not (instance? Number "x")))
  (is (instance? String "x"))
  (is (instance? Long 1))
  (is (instance? java.util.regex.Pattern #"a"))
  (is (= "java.lang.Math" (str Math)))
  (is (= "java.lang.Math" (pr-str java.lang.Math)))
  (is (= "java.util.UUID" (str java.util.UUID))))

(deftest t-catch
  (is (= "caught" (try (throw (ex-info "boom" {}))
                       (catch Exception e "caught"))))
  (is (= "caught" (try (throw (ex-info "boom" {}))
                       (catch Throwable e "caught"))))
  (is (= "caught" (try (/ 1 0)
                       (catch Throwable e "caught"))))
  (is (= "boom" (try (throw (ex-info "boom" {}))
                     (catch Exception e (.getMessage e)))))
  (is (= "caught" (try (java.util.ArrayList.)
                       (catch Exception e "caught")))))

(run-tests)
