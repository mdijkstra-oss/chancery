#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CONVERSATIONS_DIR="${1:-$SCRIPT_DIR/conversations}"

if [ ! -d "$CONVERSATIONS_DIR" ]; then
    echo "Directory not found: $CONVERSATIONS_DIR"
    exit 1
fi

for file in "$CONVERSATIONS_DIR"/*.json; do
    [ -f "$file" ] || continue
    "$SCRIPT_DIR/run-one.sh" "$file"
done

echo "════════════════════════════════════════════════════════════════"
echo "DONE"
echo "════════════════════════════════════════════════════════════════"
