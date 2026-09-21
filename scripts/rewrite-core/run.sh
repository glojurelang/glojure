#!/usr/bin/env bash
set -euo pipefail

# this script should be run from the root of the repository

# Most mutations are implemented in rewrite.clj, but some are
# implemented in this script. Number tags are removed, as go doesn't
# have boxed numbers. We also truncate core.glj at deftype.

# rewrite.clj runs under babashka, which bundles rewrite-clj, its only
# dependency. Set BB to point at a specific bb binary.

# discard file after defype for now

cd scripts/rewrite-core
"${BB:-bb}" ./rewrite.clj "../../$1" | \
    sed 's/\^Number //g' | \
    sed 's/:tag Number//g' | \
    sed 's/[[:space:]]*$//'
