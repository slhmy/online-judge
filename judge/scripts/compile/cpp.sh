#!/bin/bash

# C++17 Compilation Script for Online Judge
# Language: C++17 (g++)
# Time Factor: 1.0
# Memory Factor: 1.0

# Compiler settings
COMPILER="g++"
STANDARD="-std=c++17"
OPTIMIZATION="-O2"
EXTRA_FLAGS="-lm -DONLINE_JUDGE -pipe -fno-stack-limit"

# Source and output files
SOURCE_FILE="main.cpp"
OUTPUT_FILE="main"

# Compilation time limit (seconds)
TIME_LIMIT=30

# Memory limit for compilation (KB -> MB)
MEMORY_LIMIT_MB=512

echo "=== C++17 Compilation Script ==="
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

    # Show binary info
    if [ -f "$OUTPUT_FILE" ]; then
        ls -lh "$OUTPUT_FILE"
        file "$OUTPUT_FILE"
    fi
    exit 0
elif [ $COMPILE_EXIT_CODE -eq 124 ]; then
    echo ""
    echo "Error: Compilation timeout after $TIME_LIMIT seconds"
    exit 2
else
    echo ""
    echo "Error: Compilation failed with exit code $COMPILE_EXIT_CODE"
    echo ""
    echo "Common C++17 compilation errors:"
    echo "  - Syntax errors: Check for missing semicolons, brackets"
    echo "  - Undefined reference: Check for missing function definitions"
    echo "  - Type errors: Check for type mismatches"
    echo ""
    echo "Note: The ONLINE_JUDGE macro is defined"
    exit 1
fi