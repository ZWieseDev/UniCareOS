#!/bin/bash
# Hex-convert.sh: Convert a JSON array of bytes to a hex string.
# Usage:
#   ./Hex-convert.sh '[131, 37, 105, ...]'
#   echo '[131, 37, 105, ...]' | ./Hex-convert.sh

if [ -t 0 ] && [ $# -eq 0 ]; then
  echo "Usage: $0 '[byte array]'"
  echo "Example: $0 '[131, 37, 105, 169, 84, 99, 199, 208, 163, 124, 83, 85, 70, 13, 198, 201, 124, 231, 57, 72, 136, 42, 10, 203, 20, 242, 130, 251, 178, 47, 0, 67]'"
  exit 1
fi

# Read input (argument or stdin)
if [ $# -gt 0 ]; then
  INPUT="$1"
else
  INPUT=$(cat)
fi

# Remove brackets and spaces, split by comma
BYTES=$(echo "$INPUT" | tr -d '[] ' | tr ',' ' ')

# Convert each byte to hex and concatenate
HEX=""
for b in $BYTES; do
  printf -v h "%02x" "$b"
  HEX+=$h
done

echo "$HEX"
