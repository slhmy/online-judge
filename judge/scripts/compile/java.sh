#!/bin/bash

# Java 17 Compilation Script for Online Judge
# Language: Java 17
# Time Factor: 2.0 (startup overhead)
# Memory Factor: 1.5

# Compiler settings
COMPILER="javac"
ENCODING="-encoding UTF8"
SOURCE_VERSION="-source 17"
TARGET_VERSION="-target 17"
EXTRA_FLAGS="-Xlint:all"

# Source file (class must be named 'Main')
SOURCE_FILE="Main.java"
CLASS_FILE="Main.class"

# Runtime settings
RUNTIME="java"
JVM_HEAP="-Xmx512M"
JVM_STACK="-Xss64M"
JVM_FLAGS="-DONLINE_JUDGE=true -enableassertions"

# Compilation time limit (seconds)
TIME_LIMIT=60

echo "=== Java 17 Compilation Script ==="
echo "Compiler: $COMPILER"
echo "Source: $SOURCE_FILE"
echo "Class: $CLASS_FILE"
echo ""

# Check if source file exists
if [ ! -f "$SOURCE_FILE" ]; then
    echo "Error: Source file '$SOURCE_FILE' not found"
    echo ""
    echo "Note: Java source file must be named 'Main.java' and contain a class named 'Main'"
    exit 1
fi

# Display Java version
echo "Java version:"
java -version 2>&1 | head -1
echo "Java compiler version:"
$COMPILER -version
echo ""

# Check for class name requirement
if ! grep -q "class Main" "$SOURCE_FILE"; then
    echo "Warning: Class name might not be 'Main'"
    echo "Note: The class must be named 'Main' (not public or in a package)"
fi

# Check for package declaration (should not have one)
if grep -q "^package " "$SOURCE_FILE"; then
    echo "Warning: Package declaration found"
    echo "Note: Remove any package declaration for Online Judge"
fi

# Run compilation with timeout
echo "Compiling..."
timeout $TIME_LIMIT $COMPILER $ENCODING $SOURCE_VERSION $TARGET_VERSION $EXTRA_FLAGS "$SOURCE_FILE" 2>&1

COMPILE_EXIT_CODE=$?

if [ $COMPILE_EXIT_CODE -eq 0 ]; then
    echo ""
    echo "Compilation successful!"
    echo "Output class: $CLASS_FILE"

    if [ -f "$CLASS_FILE" ]; then
        ls -lh "$CLASS_FILE"
    fi

    echo ""
    echo "To run the program:"
    echo "  $RUNTIME $JVM_HEAP $JVM_STACK $JVM_FLAGS Main"
    exit 0
elif [ $COMPILE_EXIT_CODE -eq 124 ]; then
    echo "Error: Compilation timeout after $TIME_LIMIT seconds"
    exit 2
else
    echo ""
    echo "Error: Compilation failed with exit code $COMPILE_EXIT_CODE"
    echo ""
    echo "Common Java compilation errors:"
    echo "  - Class name mismatch: Ensure class is named 'Main'"
    echo "  - Package declaration: Remove any package statement"
    echo "  - Missing imports: Add required import statements"
    echo "  - Syntax errors: Check for missing braces, semicolons"
    exit 1
fi