#!/bin/bash

# Script to download font files from imap.mn example
# This downloads the commonly used font ranges for Noto Sans Regular

FONT_DIR="./public/fonts/Noto Sans Regular"
BASE_URL="https://imap.mn/06826032-4372-11ec-81d3-0242ac130003/fonts/Noto%20Sans%20Regular"

# Create font directory
mkdir -p "$FONT_DIR"

# Common Unicode ranges for Noto Sans Regular
# These cover most Latin, Cyrillic, and common symbols
RANGES=(
  "0-255"       # Basic Latin + Latin-1 Supplement
  "256-511"     # Latin Extended-A + Latin Extended-B
  "512-767"     # IPA Extensions + Spacing Modifier Letters
  "768-1023"    # Combining Diacritical Marks + Greek and Coptic
  "1024-1279"   # Cyrillic + Cyrillic Supplement
  "1280-1535"   # Armenian + Hebrew
  "1536-1791"   # Arabic
  "11264-11519" # Glagolitic + Latin Extended-C
  "19968-20223" # CJK Unified Ideographs (partial)
  "20224-20479" # CJK Unified Ideographs (partial)
)

echo "Downloading Noto Sans Regular font files..."

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
echo "Font download complete!"
echo "Fonts saved to: $FONT_DIR"
