#!/bin/bash

# Script to download Noto Sans Bold font files

FONT_DIR="./public/fonts/Noto Sans Bold"
BASE_URL="https://imap.mn/06826032-4372-11ec-81d3-0242ac130003/fonts/Noto%20Sans%20Bold"

# Create font directory
mkdir -p "$FONT_DIR"

# Common Unicode ranges
RANGES=(
  "0-255"
  "256-511"
  "512-767"
  "768-1023"
  "1024-1279"
  "1280-1535"
  "1536-1791"
  "11264-11519"
  "19968-20223"
  "20224-20479"
)

echo "Downloading Noto Sans Bold font files..."

for range in "${RANGES[@]}"; do
  echo "Downloading range: $range"
  curl -s "${BASE_URL}/${range}.pbf" -o "${FONT_DIR}/${range}.pbf"
  
  if [ $? -eq 0 ]; then
    echo "✓ Downloaded ${range}.pbf"
  else
    echo "✗ Failed to download ${range}.pbf"
  fi
done

echo ""
echo "Bold Font download complete!"
