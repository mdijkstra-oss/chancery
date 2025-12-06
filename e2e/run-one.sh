#!/bin/bash
set -e

SERVER_URL="${SERVER_URL:-http://localhost:8081}"
INPUT_FILE="$1"

if [ -z "$INPUT_FILE" ]; then
    echo "Usage: $0 <input-file.json>"
    exit 1
fi

if [ ! -f "$INPUT_FILE" ]; then
    echo "File not found: $INPUT_FILE"
    exit 1
fi

FILENAME=$(basename "$INPUT_FILE")

echo "════════════════════════════════════════════════════════════════"
echo "FILE: $FILENAME"
echo "════════════════════════════════════════════════════════════════"
echo ""
echo "──── INPUT ────"
cat "$INPUT_FILE" | jq .
echo ""
echo "──── OUTPUT ────"
curl -sN "${SERVER_URL}/chat" \
    -H "Content-Type: application/json" \
    -d "$(jq -n --argjson msgs "$(cat "$INPUT_FILE")" '{messages: $msgs}')"
echo ""
echo ""
