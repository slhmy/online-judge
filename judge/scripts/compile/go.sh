#!/bin/bash

# Go 1.21 Compilation Script for Online Judge
# Language: Go 1.21
# Time Factor: 1.0
# Memory Factor: 1.0

# Compiler settings
COMPILER="go"
BUILD_CMD="build"
LDFLAGS="-ldflags -s -w"  # Strip debug info for smaller binary

# Source and output files
SOURCE_FILE="main.go"
OUTPUT_FILE="main"

# Compilation time limit (seconds)
TIME_LIMIT=60

echo "=== Go 1.21 Compilation Script ==="
echo "Compiler: $COMPILER"
echo "Source: $SOURCE_FILE"
echo "Output: $OUTPUT_FILE"
echo ""

# Check if source file exists
if [ ! -f "$SOURCE_FILE" ]; then
    echo "Error: Source file '$SOURCE_FILE' not found"
    exit 1
fi

# Display Go version
echo "Go version:"
$COMPILER version
echo ""

# Check for package declaration
if ! grep -q "^package " "$SOURCE_FILE"; then
    echo "Warning: Missing package declaration"
    echo "Note: Go source must start with a package declaration (usually 'package main')"
fi

# Check for main function
if ! grep -q "func main()" "$SOURCE_FILE"; then
    echo "Warning: Missing main function"
    echo "Note: Executable Go programs must have a main function"
fi

# Run compilation with timeout
echo "Compiling..."
timeout $TIME_LIMIT $COMPILER $BUILD_CMD -o "$OUTPUT_FILE" $LDFLAGS "$SOURCE_FILE" 2>&1

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
    echo "Common Go compilation errors:"
    echo "  - undefined: Variable/function not defined"
    echo "  - syntax error: Check for syntax issues"
    echo "  - imported but not used: Remove unused imports"
    echo "  - declared but not used: Remove unused variables"
    exit 1
fi