# this script should be run from the root of the repository

# Most mutations are implemented in rewrite.clj, but some are
# implemented in this script. Number tags are removed, as go doesn't
# have boxed numbers. We also truncate core.glj at deftype.

# discard file after defype for now

cd scripts/rewrite-core
clj -M ./rewrite.clj "../../$1" | \
    sed 's/\^Number //g' | \
    sed 's/:tag Number//g' | \
    sed 's/clojure/glojure/g'
