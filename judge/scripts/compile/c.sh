#!/bin/bash

# Enhanced C11 Compilation Script for Online Judge
# Language: C11 (gcc)
# Time Factor: 1.0 (native speed)
# Memory Factor: 1.0 (standard memory)
# Features: Optimized for competitive programming

set -e

# ==================== Configuration ====================
COMPILER="gcc"
STANDARD="-std=c11"
OPTIMIZATION="-O2"  # Good balance between speed and compilation time
EXTRA_FLAGS="-lm -DONLINE_JUDGE -pipe"

# Security and warning flags
WERROR_FLAGS="-Wall -Wextra -Wno-unused-result"

# Source and output files
SOURCE_FILE="main.c"
OUTPUT_FILE="main"

# Compilation resource limits
TIME_LIMIT=30        # seconds
MEMORY_LIMIT_MB=512  # megabytes

# Exit codes
EXIT_SUCCESS=0
EXIT_COMPILE_ERROR=1
EXIT_TIMEOUT=2
EXIT_FILE_ERROR=4

# ==================== Helper Functions ====================

parse_c_error() {
    local output="$1"
    local error_type=""
    local line_num=""
    local message=""

    # Extract structured error information
    if echo "$output" | grep -qE "main\.c:\d+:\d+: error:"; then
        error_type="error"
        line_num=$(echo "$output" | grep -oE "main\.c:\d+" | grep -oE "\d+" | head -1)
        message=$(echo "$output" | grep -oE "error: .*" | head -1 | sed 's/error: //')
    elif echo "$output" | grep -qE "main\.c:\d+: error:"; then
        error_type="error"
        line_num=$(echo "$output" | grep -oE "main\.c:\d+" | grep -oE "\d+" | head -1)
        message=$(echo "$output" | grep -oE "error: .*" | head -1 | sed 's/error: //')
    elif echo "$output" | grep -qE "^error:"; then
        error_type="error"
        message=$(echo "$output" | grep -oE "^error: .*" | sed 's/error: //')
    fi

    if [ -n "$line_num" ]; then
        echo "Error at line $line_num: $message"
    elif [ -n "$message" ]; then
        echo "Error: $message"
    fi
}

show_c_common_errors() {
    echo ""
    echo "Common C11 compilation errors and solutions:"
    echo ""
    echo "  [Syntax Errors]"
    echo "    - Missing semicolon: Add ';' after statements"
    echo "    - Missing brackets: Balance {} [] ()"
    echo "    - Unexpected token: Check for typos or invalid syntax"
    echo "    - Declaration missing: Add type or function declaration"
    echo ""
    echo "  [Type Errors]"
    echo "    - Implicit declaration: Include header or declare function"
    echo "    - Incompatible types: Use explicit type conversion"
    echo "    - Invalid type: Check type definition"
    echo ""
    echo "  [Undefined References]"
    echo "    - Undefined reference to: Implement the function or link library"
    echo "    - Was not declared: Add declaration or include header"
    echo "    - Linker error: Check function implementation"
    echo ""
    echo "  [Warning Flags]"
    echo "    - Unused variable: Remove or use the variable"
    echo "    - Unused parameter: Mark as unused or remove"
    echo "    - Implicit conversion: Use explicit cast"
    echo ""
}

check_c_source() {
    local source="$1"
    local warnings=""

    # Check for common issues
    if ! grep -qE "^#include" "$source"; then
        warnings+="Warning: No include directives found\n"
    fi

    # Check for standard headers
    if grep -qE "printf|scanf" "$source" && ! grep -qE "#include <stdio.h>"; then
        warnings+="Warning: printf/scanf used but stdio.h not included\n"
    fi

    # Check for malloc usage
    if grep -qE "malloc|free|calloc|realloc" "$source" && ! grep -qE "#include <stdlib.h>"; then
        warnings+="Warning: memory functions used but stdlib.h not included\n"
    fi

    # Check for math functions
    if grep -qE "sin|cos|sqrt|pow|fabs" "$source" && ! grep -qE "#include <math.h>"; then
        warnings+="Warning: math functions used but math.h not included\n"
    fi

    # Check for potential overflow issues
    if grep -qE "int\s+\w+\s*=\s*\d{10,}" "$source"; then
        warnings+="Warning: Large integer literal assigned to int - consider long\n"
    fi

    # Check for unchecked return values
    if grep -qE "scanf\(" "$source" && ! grep -qE "scanf\(.*\)\s*==\s*\d"; then
        warnings+="Tip: Check scanf return value for robust input handling\n"
    fi

    if [ -n "$warnings" ]; then
        echo "$warnings"
    fi
}

# ==================== Main Compilation ====================

echo "=========================================="
echo "  C11 Compilation Script"
echo "  Online Judge Sandbox Environment"
echo "=========================================="
echo ""
echo "Compiler: $COMPILER"
echo "Standard: $STANDARD"
echo "Optimization: $OPTIMIZATION"
echo "Source: $SOURCE_FILE"
echo "Output: $OUTPUT_FILE"
echo "Time Limit: $TIME_LIMIT seconds"
echo "Memory Limit: $MEMORY_LIMIT_MB MB"
echo ""

# Check if source file exists
if [ ! -f "$SOURCE_FILE" ]; then
    echo "Error: Source file '$SOURCE_FILE' not found"
    echo ""
    echo "Expected file structure:"
    echo "  - main.c (required) - Your C source code"
    echo ""
    exit $EXIT_FILE_ERROR
fi

# Display compiler version
echo "Compiler version:"
$COMPILER --version | head -1
echo ""

# Pre-compilation source check
echo "Pre-compilation checks..."
check_c_source "$SOURCE_FILE"
echo ""

# Run compilation with timeout and memory limit
echo "Compiling..."
COMPILER_OUTPUT=$(timeout $TIME_LIMIT $COMPILER $STANDARD $OPTIMIZATION $EXTRA_FLAGS $WERROR_FLAGS -o "$OUTPUT_FILE" "$SOURCE_FILE" 2>&1)
COMPILE_EXIT_CODE=$?

if [ $COMPILE_EXIT_CODE -eq 0 ]; then
    echo ""
    echo "=========================================="
    echo "  COMPILATION SUCCESSFUL"
    echo "=========================================="
    echo ""
    echo "Output binary: $OUTPUT_FILE"

    if [ -f "$OUTPUT_FILE" ]; then
        BINARY_SIZE=$(ls -lh "$OUTPUT_FILE" | awk '{print $5}')
        BINARY_TYPE=$(file "$OUTPUT_FILE" | cut -d: -f2)
        echo "Binary size: $BINARY_SIZE"
        echo "Binary type:$BINARY_TYPE"
    fi

    echo ""
    echo "To run the program:"
    echo "  ./$OUTPUT_FILE"
    echo ""
    echo "Runtime notes:"
    echo "  - ONLINE_JUDGE macro is defined"
    echo "  - Optimized with -O2"
    echo "  - Time factor: 1.0 (native speed)"
    echo "  - Memory factor: 1.0 (standard memory)"
    echo ""
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
    echo "  - Complex code structure"
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
    PARSED_ERROR=$(parse_c_error "$COMPILER_OUTPUT")
    if [ -n "$PARSED_ERROR" ]; then
        echo "Parsed error:"
        echo "$PARSED_ERROR"
    fi

    # Show common error help
    show_c_common_errors

    exit $EXIT_COMPILE_ERROR
fi