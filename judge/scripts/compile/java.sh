#!/bin/bash

# Enhanced Java 17 Compilation Script for Online Judge
# Language: Java 17 (OpenJDK)
# Time Factor: 2.0 (JVM startup overhead)
# Memory Factor: 1.5 (JVM memory overhead)
# Features: Optimized for competitive programming

set -e

# ==================== Configuration ====================
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

# Compilation resource limits
TIME_LIMIT=60        # seconds (Java compilation can be slow)
MEMORY_LIMIT_MB=1024 # megabytes

# Exit codes
EXIT_SUCCESS=0
EXIT_COMPILE_ERROR=1
EXIT_TIMEOUT=2
EXIT_CLASS_NAME_ERROR=3
EXIT_FILE_ERROR=4

# ==================== Helper Functions ====================

parse_java_error() {
    local output="$1"
    local error_type=""
    local line_num=""
    local message=""

    # Parse structured Java errors
    if echo "$output" | grep -qE "Main\.java:\d+: error:"; then
        error_type="error"
        line_num=$(echo "$output" | grep -oE "Main\.java:\d+" | grep -oE "\d+" | head -1)
        message=$(echo "$output" | grep -oE "error: .*" | head -1 | sed 's/error: //')
    elif echo "$output" | grep -qE "^error:"; then
        error_type="error"
        message=$(echo "$output" | grep -oE "^error: .*" | sed 's/error: //')
    fi

    # Parse common specific errors
    if echo "$output" | grep -qE "cannot find symbol"; then
        symbol=$(echo "$output" | grep -oE "symbol:\s+\w+" | sed 's/symbol:\s*//')
        location=$(echo "$output" | grep -oE "location:\s+\w+" | sed 's/location:\s*//')
        if [ -n "$symbol" ]; then
            message="Cannot find symbol: $symbol (in $location)"
        fi
    fi

    if echo "$output" | grep -qE "class Main is public, should be declared"; then
        message="Class Main must be in file Main.java"
    fi

    if [ -n "$error_type" ]; then
        if [ -n "$line_num" ]; then
            echo "$error_type at line $line_num: $message"
        else
            echo "$error_type: $message"
        fi
    fi
}

show_java_common_errors() {
    echo ""
    echo "Common Java 17 compilation errors and solutions:"
    echo ""
    echo "  [Class/Package Errors]"
    echo "    - Class must be named 'Main': Rename your class to Main"
    echo "    - Package declaration: Remove any package statement"
    echo "    - Public class issue: File name must match public class name"
    echo ""
    echo "  [Symbol Errors]"
    echo "    - Cannot find symbol: Import required class or define method"
    echo "    - Variable not found: Declare variable before use"
    echo "    - Method not found: Check method signature and parameters"
    echo ""
    echo "  [Type Errors]"
    echo "    - Incompatible types: Use explicit conversion"
    echo "    - Cannot be applied: Check argument types match method signature"
    echo "    - Required type: Add generic type parameter"
    echo ""
    echo "  [Syntax Errors]"
    echo "    - Missing semicolon: Add ';' after statements"
    echo "    - Unclosed string literal: Close string with matching quote"
    echo "    - Illegal start of expression: Check for syntax issues"
    echo ""
}

check_java_source() {
    local source="$1"
    local warnings=""
    local errors=""

    # Check for class name requirement (MUST be Main for Online Judge)
    if ! grep -qE "class Main" "$source"; then
        errors+="Error: Class must be named 'Main'\n"
    fi

    # Check for public class Main
    if grep -qE "public class (?!Main)" "$source"; then
        errors+="Error: Only 'Main' class can be public\n"
    fi

    # Check for package declaration (should not have one)
    if grep -qE "^package " "$source"; then
        errors+="Error: Remove package declaration\n"
    fi

    # Check for common imports
    if grep -qE "Scanner|ArrayList|HashMap|Arrays|Collections" "$source" && ! grep -qE "import java\."; then
        warnings+="Warning: Standard library classes used but no imports found\n"
    fi

    # Check for BufferedReader usage (performance hint)
    if ! grep -qE "BufferedReader|Scanner" "$source" && ! grep -qE "System.in|System.out"; then
        warnings+="Note: No I/O detected - may need Scanner or BufferedReader\n"
    fi

    if grep -qE "Scanner" "$source" && ! grep -qE "BufferedReader"; then
        warnings+="Tip: BufferedReader is faster for large input\n"
    fi

    if [ -n "$errors" ]; then
        echo "$errors"
        return 1
    fi

    if [ -n "$warnings" ]; then
        echo "$warnings"
    fi
    return 0
}

show_java_performance_tips() {
    echo ""
    echo "Java 17 Performance Tips for Competitive Programming:"
    echo ""
    echo "  [Input Optimization]"
    echo "    // Fast input using BufferedReader"
    echo "    BufferedReader br = new BufferedReader(new InputStreamReader(System.in));"
    echo "    String[] parts = br.readLine().split(\" \");"
    echo ""
    echo "  [Output Optimization]"
    echo "    // Use StringBuilder for large output"
    echo "    StringBuilder sb = new StringBuilder();"
    echo "    sb.append(result).append(\"\\n\");"
    echo "    System.out.print(sb);"
    echo ""
    echo "  [Data Structures]"
    echo "    - ArrayList: Dynamic array, O(1) append, O(n) insert"
    echo "    - HashMap: O(1) average lookup/insert"
    echo "    - TreeSet: Sorted set, O(log n) operations"
    echo "    - PriorityQueue: Min/max heap, O(log n) insert/remove"
    echo ""
    echo "  [Memory Management]"
    echo "    - Set initial capacities when possible"
    echo "    - Avoid creating objects in loops"
    echo "    - Reuse StringBuilder instead of creating new ones"
    echo ""
    echo "  [Standard Template]"
    echo "    import java.util.*;"
    echo "    import java.io.*;"
    echo "    public class Main {"
    echo "        public static void main(String[] args) throws IOException {"
    echo "            BufferedReader br = new BufferedReader(new InputStreamReader(System.in));"
    echo "            // Your solution here"
    echo "        }"
    echo "    }"
    echo ""
}

# ==================== Main Compilation ====================

echo "=========================================="
echo "  Java 17 Compilation Script"
echo "  Online Judge Sandbox Environment"
echo "=========================================="
echo ""
echo "Compiler: $COMPILER"
echo "Source Version: $SOURCE_VERSION"
echo "Target Version: $TARGET_VERSION"
echo "Source: $SOURCE_FILE"
echo "Class: $CLASS_FILE"
echo "Time Limit: $TIME_LIMIT seconds"
echo "Memory Limit: $MEMORY_LIMIT_MB MB"
echo ""

# Check if source file exists
if [ ! -f "$SOURCE_FILE" ]; then
    echo "Error: Source file '$SOURCE_FILE' not found"
    echo ""
    echo "Expected file structure:"
    echo "  - Main.java (required) - Your Java source code"
    echo "  - Class name MUST be 'Main' (not public or in a package)"
    echo ""
    exit $EXIT_FILE_ERROR
fi

# Display Java version
echo "Java version:"
java -version 2>&1 | head -1
echo "Java compiler version:"
$COMPILER -version
echo ""

# Pre-compilation source check
echo "Pre-compilation checks..."
if ! check_java_source "$SOURCE_FILE"; then
    echo ""
    echo "=========================================="
    echo "  PRE-COMPILATION CHECK FAILED"
    echo "=========================================="
    echo ""
    echo "Fix the above errors before compilation"
    show_java_common_errors
    exit $EXIT_CLASS_NAME_ERROR
fi
echo ""

# Run compilation with timeout
echo "Compiling..."
COMPILER_OUTPUT=$(timeout $TIME_LIMIT $COMPILER $ENCODING $SOURCE_VERSION $TARGET_VERSION $EXTRA_FLAGS "$SOURCE_FILE" 2>&1)
COMPILE_EXIT_CODE=$?

if [ $COMPILE_EXIT_CODE -eq 0 ]; then
    echo ""
    echo "=========================================="
    echo "  COMPILATION SUCCESSFUL"
    echo "=========================================="
    echo ""
    echo "Output class: $CLASS_FILE"

    if [ -f "$CLASS_FILE" ]; then
        CLASS_SIZE=$(ls -lh "$CLASS_FILE" | awk '{print $5}')
        echo "Class size: $CLASS_SIZE"
    fi

    echo ""
    echo "To run the program:"
    echo "  $RUNTIME $JVM_HEAP $JVM_STACK $JVM_FLAGS Main"
    echo ""
    echo "Runtime notes:"
    echo "  - Heap limit: 512MB (-Xmx512M)"
    echo "  - Stack size: 64MB (-Xss64M)"
    echo "  - ONLINE_JUDGE property set"
    echo "  - Assertions enabled"
    echo "  - Time factor: 2.0 (JVM startup overhead)"
    echo ""

    show_java_performance_tips

    exit $EXIT_SUCCESS

elif [ $COMPILE_EXIT_CODE -eq 124 ]; then
    echo ""
    echo "=========================================="
    echo "  COMPILATION TIMEOUT"
    echo "=========================================="
    echo ""
    echo "Error: Compilation exceeded $TIME_LIMIT seconds"
    echo ""
    echo "Possible causes:"
    echo "  - Very large source file"
    echo "  - Complex generic types"
    echo "  - Deeply nested code"
    echo ""
    exit $EXIT_TIMEOUT

else
    echo ""
    echo "=========================================="
    echo "  COMPILATION FAILED"
    echo "=========================================="
    echo ""
    echo "Compiler output:"
    echo "$COMPILER_OUTPUT"
    echo ""

    # Parse and show structured error
    PARSED_ERROR=$(parse_java_error "$COMPILER_OUTPUT")
    if [ -n "$PARSED_ERROR" ]; then
        echo "Parsed error:"
        echo "$PARSED_ERROR"
    fi

    # Show common error help
    show_java_common_errors

    exit $EXIT_COMPILE_ERROR
fi