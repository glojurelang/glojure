(ns codegen.test.jvm-interop)

;; AOT twin of test/glojure/test_glojure/jvm_interop.clj. Each check
;; returns a keyword naming itself on failure so the mismatch is visible.

(defn- check [name ok] (when-not ok name))

(defn ^{:expected-output :ok} -main []
  (or
   (check :math-abs (= 3 (Math/abs -3)))
   (check :parse-boolean (= true (Boolean/parseBoolean "true")))
   (check :parse-int (= 42 (Integer/parseInt "42")))
   (check :parse-long (= 42 (Long/parseLong "42")))
   (check :parse-double (= 1.5 (Double/parseDouble "1.5")))
   (check :string-value-of (= "7" (String/valueOf 7)))
   (check :is-digit (Character/isDigit \7))
   (check :fq-math-abs (= 3 (java.lang.Math/abs -3)))
   (check :fq-parse-boolean (= true (java.lang.Boolean/parseBoolean "true")))
   (check :fq-parse-int (= 42 (java.lang.Integer/parseInt "42")))
   (check :fq-pattern-matches (java.util.regex.Pattern/matches "a+" "aaa"))
   (check :fq-pattern-quote (= "a\\.b" (java.util.regex.Pattern/quote "a.b")))
   (check :integer-max (= 2147483647 Integer/MAX_VALUE))
   (check :long-min (= -9223372036854775808 Long/MIN_VALUE))
   (check :fq-integer-max (= 2147483647 java.lang.Integer/MAX_VALUE))
   (check :double-nan (Double/isNaN Double/NaN))
   (check :fq-pattern-flag (= 2 java.util.regex.Pattern/CASE_INSENSITIVE))
   (check :integer-ctor (= 42 (Integer. "42")))
   (check :long-ctor-str (= 42 (Long. "42")))
   (check :long-ctor-num (= 7 (Long. 7)))
   (check :double-ctor (= 1.5 (Double. "1.5")))
   (check :character-ctor (= \a (Character. \a)))
   (check :boolean-ctor (= true (Boolean. "true")))
   (check :pattern-ctor (= "a+" (.pattern (java.util.regex.Pattern. "a+"))))
   (check :pattern-compile (= "a+" (.pattern (java.util.regex.Pattern/compile "a+"))))
   (check :to-upper (= "ABC" (.toUpperCase "abc")))
   (check :length (= 3 (.length "abc")))
   (check :substring (= "b,c" (.substring "a,b,c" 2)))
   (check :trim (= "x" (.trim "  x  ")))
   (check :get-bytes (= [97 98] (vec (.getBytes "ab"))))
   (check :number-long (instance? Number 1))
   (check :number-double (instance? Number 1.5))
   (check :number-string (not (instance? Number "x")))
   (check :string-class (instance? String "x"))
   (check :long-class (instance? Long 1))
   (check :pattern-class (instance? java.util.regex.Pattern #"a"))
   (check :math-name (= "java.lang.Math" (str Math)))
   (check :fq-math-name (= "java.lang.Math" (pr-str java.lang.Math)))
   (check :uuid-name (= "java.util.UUID" (str java.util.UUID)))
   (check :catch-exception
          (= "caught" (try (throw (ex-info "boom" {}))
                           (catch Exception e "caught"))))
   (check :catch-throwable
          (= "caught" (try (throw (ex-info "boom" {}))
                           (catch Throwable e "caught"))))
   (check :catch-throwable-panic
          (= "caught" (try (/ 1 0) (catch Throwable e "caught"))))
   (check :get-message
          (= "boom" (try (throw (ex-info "boom" {}))
                         (catch Exception e (.getMessage e)))))
   (check :unknown-class
          (= "caught" (try (java.util.ArrayList.)
                           (catch Exception e "caught"))))
   :ok))
