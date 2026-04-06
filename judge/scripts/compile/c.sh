#!/bin/bash

# C11 Compilation Script for Online Judge
# Language: C11 (gcc)
# Time Factor: 1.0
# Memory Factor: 1.0

# Compiler settings
COMPILER="gcc"
STANDARD="-std=c11"
OPTIMIZATION="-O2"
EXTRA_FLAGS="-lm -DONLINE_JUDGE -pipe"

# Source and output files
SOURCE_FILE="main.c"
OUTPUT_FILE="main"

# Compilation time limit (seconds)
TIME_LIMIT=30

echo "=== C11 Compilation Script ==="
echo "Compiler: $COMPILER"
echo "Standard: $STANDARD"
echo "Source: $SOURCE_FILE"
echo "Output: $OUTPUT_FILE"
echo ""

# Check if source file exists
if [ ! -f "$SOURCE_FILE" ]; then
    echo "Error: Source file '$SOURCE_FILE' not found"
    exit 1
fi

# Display compiler version
echo "Compiler version:"
$COMPILER --version | head -1
echo ""

# Run compilation with timeout
echo "Compiling..."
timeout $TIME_LIMIT $COMPILER $STANDARD $OPTIMIZATION $EXTRA_FLAGS -o "$OUTPUT_FILE" "$SOURCE_FILE" 2>&1

COMPILE_EXIT_CODE=$?

if [ $COMPILE_EXIT_CODE -eq 0 ]; then
    echo ""
    echo "Compilation successful!"
    echo "Output binary: $OUTPUT_FILE"

    if [ -f "$OUTPUT_FILE" ]; then
        ls -lh "$OUTPUT_FILE"
        file "$OUTPUT_FILE"
    fi
    exit 0
elif [ $COMPILE_EXIT_CODE -eq 124 ]; then
    echo "Error: Compilation timeout after $TIME_LIMIT seconds"
    exit 2
else
    echo ""
    echo "Error: Compilation failed with exit code $COMPILE_EXIT_CODE"
    exit 1
fi