# discard file after defype for now
clj -M ./rewrite.clj $1 | sed '/deftype/Q'
