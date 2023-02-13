# discard file after defype for now
clj -M ./rewrite.clj | sed '/deftype/Q'
