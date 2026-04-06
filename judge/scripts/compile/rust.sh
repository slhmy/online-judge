#!/bin/bash

# Rust Compilation Script for Online Judge
# Language: Rust (stable)
# Time Factor: 1.0
# Memory Factor: 1.0

# Compiler settings
COMPILER="rustc"
OPTIMIZATION="-O"

# Source and output files
SOURCE_FILE="main.rs"
OUTPUT_FILE="main"

# Compilation time limit (seconds) - Rust can be slow
TIME_LIMIT=120

echo "=== Rust Compilation Script ==="
echo "Compiler: $COMPILER"
echo "Source: $SOURCE_FILE"
echo "Output: $OUTPUT_FILE"
echo ""

# Check if source file exists
if [ ! -f "$SOURCE_FILE" ]; then
    echo "Error: Source file '$SOURCE_FILE' not found"
    exit 1
fi

# Display Rust version
echo "Rust version:"
$COMPILER --version
cargo --version 2>/dev/null || true
echo ""

# Check for main function
if ! grep -q "fn main()" "$SOURCE_FILE"; then
    echo "Warning: Missing main function"
    echo "Note: Executable Rust programs must have a main function"
fi

# Run compilation with timeout
echo "Compiling..."
timeout $TIME_LIMIT $COMPILER $OPTIMIZATION -o "$OUTPUT_FILE" "$SOURCE_FILE" 2>&1

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
    echo ""
    echo "Common Rust compilation errors:"
    echo "  - error[E0425]: cannot find value/function"
    echo "  - error[E0308]: mismatched types"
    echo "  - error[E0599]: no method available"
    echo "  - expected vs found type mismatch"
    exit 1
fi