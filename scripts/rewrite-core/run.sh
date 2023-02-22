# this script should be run from the root of the repository
# discard file after defype for now

cd scripts/rewrite-core
clj -M ./rewrite.clj "../../$1" | sed '/deftype/Q'
