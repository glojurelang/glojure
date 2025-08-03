#!/bin/bash

# Script to update Clojure source files from a specific GitHub tag
# Usage: ./update-clojure-sources.sh <tag>
# Example: ./update-clojure-sources.sh clojure-1.12.1

set -euo pipefail

# Check if tag argument is provided
if [ $# -eq 0 ]; then
    echo "Error: Please provide a Clojure tag as an argument"
    echo "Usage: $0 <tag>"
    echo "Example: $0 clojure-1.12.1"
    exit 1
fi

TAG=$1
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ORIGINALS_DIR="${SCRIPT_DIR}/originals"
BASE_URL="https://raw.githubusercontent.com/clojure/clojure/${TAG}/src/clj/clojure"

# List of files to download (based on current files in originals/)
FILES=(
    "core.clj"
    "core_print.clj"
    "template.clj"
    "test.clj"
    "uuid.clj"
    "walk.clj"
)

echo "Updating Clojure source files from tag: ${TAG}"
echo "Destination: ${ORIGINALS_DIR}"
echo

# Ensure originals directory exists
mkdir -p "${ORIGINALS_DIR}"

# Download each file
for file in "${FILES[@]}"; do
    url="${BASE_URL}/${file}"
    dest="${ORIGINALS_DIR}/${file}"
    
    echo -n "Downloading ${file}... "
    
    if curl -fsSL "${url}" -o "${dest}"; then
        echo "✓"
    else
        echo "✗"
        echo "Error: Failed to download ${file} from ${url}"
        exit 1
    fi
done

echo
echo "Successfully updated ${#FILES[@]} files from Clojure ${TAG}"