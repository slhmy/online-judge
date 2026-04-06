#!/bin/bash

# Enhanced Go 1.21 Compilation Script for Online Judge
# Language: Go 1.21
# Time Factor: 1.0 (native speed)
# Memory Factor: 1.0 (standard memory)
# Features: Optimized for competitive programming

set -e

# ==================== Configuration ====================
COMPILER="go"
BUILD_CMD="build"
# Strip debug info for smaller binary (-s: strip symbol table, -w: strip DWARF)
LDFLAGS="-ldflags -s -w"

# Source and output files
SOURCE_FILE="main.go"
OUTPUT_FILE="main"

# Compilation resource limits
TIME_LIMIT=60        # seconds
MEMORY_LIMIT_MB=512  # megabytes

# Exit codes
EXIT_SUCCESS=0
EXIT_COMPILE_ERROR=1
EXIT_TIMEOUT=2
EXIT_PACKAGE_ERROR=3
EXIT_FILE_ERROR=4

# ==================== Helper Functions ====================

parse_go_error() {
    local output="$1"
    local error_type=""
    local line_num=""
    local col_num=""
    local message=""

    # Parse structured Go errors
    if echo "$output" | grep -qE "main\.go:\d+:\d+: "; then
        error_type="error"
        line_num=$(echo "$output" | grep -oE "main\.go:\d+" | grep -oE "\d+" | head -1)
        col_num=$(echo "$output" | grep -oE ":\d+:" | grep -oE "\d+" | tail -1)
        message=$(echo "$output" | grep -oE "main\.go:\d+:\d+: .*" | sed 's/main\.go:\d+:\d+: //' | head -1)
    elif echo "$output" | grep -qE "main\.go:\d+: "; then
        error_type="error"
        line_num=$(echo "$output" | grep -oE "main\.go:\d+" | grep -oE "\d+" | head -1)
        message=$(echo "$output" | grep -oE "main\.go:\d+: .*" | sed 's/main\.go:\d+: //' | head -1)
    fi

    # Parse undefined errors
    if echo "$output" | grep -qE "undefined:"; then
        message=$(echo "$output" | grep -oE "undefined: .*" | sed 's/undefined: /Undefined: /')
    fi

    # Parse syntax errors
    if echo "$output" | grep -qE "syntax error:"; then
        message=$(echo "$output" | grep -oE "syntax error: .*" | sed 's/syntax error: /Syntax error: /')
    fi

    if [ -n "$error_type" ]; then
        if [ -n "$line_num" ] && [ -n "$col_num" ]; then
            echo "$error_type at line $line_num:$col_num: $message"
        elif [ -n "$line_num" ]; then
            echo "$error_type at line $line_num: $message"
        else
            echo "$error_type: $message"
        fi
    fi
}

show_go_common_errors() {
    echo ""
    echo "Common Go 1.21 compilation errors and solutions:"
    echo ""
    echo "  [Package Errors]"
    echo "    - Missing package declaration: Add 'package main' at top"
    echo "    - Import used but not declared: Add import statement"
    echo "    - Imported but not used: Remove unused import"
    echo ""
    echo "  [Function Errors]"
    echo "    - Missing main function: Add 'func main() {}'"
    echo "    - Undefined function: Define function or import package"
    echo "    - Too many arguments: Check function signature"
    echo ""
    echo "  [Type Errors]"
    echo "    - Cannot use X as type Y: Use type conversion"
    echo "    - Assignment mismatch: Match number of values"
    echo "    - Invalid operation: Check type compatibility"
    echo ""
    echo "  [Variable Errors]"
    echo "    - Declared but not used: Remove or use the variable"
    echo "    - Undefined: Define variable before use"
    echo "    - No new variables on left side: Use = instead of := for reassignment"
    echo ""
    echo "  [Syntax Errors]"
    echo "    - Unexpected token: Check for syntax issues"
    echo "    - Missing braces: Balance {}"
    echo "    - Non-declaration statement: Move declaration outside function"
    echo ""
}

check_go_source() {
    local source="$1"
    local warnings=""
    local errors=""

    # Check for package declaration (required)
    if ! grep -qE "^package "; then
        errors+="Error: Missing package declaration (must be 'package main')\n"
    fi

    # Check package is 'main' (required for executable)
    if ! grep -qE "^package main"; then
        errors+="Error: Package must be 'main' for executable\n"
    fi

    # Check for main function (required)
    if ! grep -qE "func main\("; then
        errors+="Error: Missing main function\n"
    fi

    # Check for common issues
    if grep -qE "fmt.Print" "$source" && ! grep -qE "fmt" "$source" | grep import; then
        warnings+="Warning: fmt.Print used but fmt package may not be imported\n"
    fi

    # Check for potential performance issues
    if grep -qE "for.*range" "$source" && grep -qE "append\(.*\*"; then
        warnings+="Tip: Preallocate slice capacity when possible\n"
    fi

    if grep -qE "strings.Split" "$source"; then
        warnings+="Note: strings.Split creates slice - consider bytes for large data\n"
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

show_go_performance_tips() {
    echo ""
    echo "Go 1.21 Performance Tips for Competitive Programming:"
    echo ""
    echo "  [Input Optimization]"
    echo "    // Fast input using bufio"
    echo "    scanner := bufio.NewScanner(os.Stdin)"
    echo "    scanner.Split(bufio.ScanWords)"
    echo "    // scanner.Scan() reads next token"
    echo ""
    echo "  [Output Optimization]"
    echo "    // Buffered output"
    echo "    w := bufio.NewWriter(os.Stdout)"
    echo "    defer w.Flush()"
    echo "    fmt.Fprintln(w, result)"
    echo ""
    echo "  [Data Structures]"
    echo "    - slice: Dynamic array, append() for growth"
    echo "    - map: Hash table, O(1) average lookup"
    echo "    - container/list: Linked list (use sparingly)"
    echo "    - container/heap: Heap interface implementation"
    echo ""
    echo "  [Memory Optimization]"
    echo "    - Preallocate slice: make([]int, 0, n)"
    echo "    - Reuse buffers instead of creating new ones"
    echo "    - Use pointers for large structs"
    echo ""
    echo "  [Standard Template]"
    echo "    package main"
    echo "    import ("
    echo "        \"bufio\""
    echo "        \"fmt\""
    echo "        \"os\""
    echo "        \"strconv\""
    echo "    )"
    echo "    func main() {"
    echo "        scanner := bufio.NewScanner(os.Stdin)"
    echo "        // Your solution here"
    echo "    }"
    echo ""
}

# ==================== Main Compilation ====================

echo "=========================================="
echo "  Go 1.21 Compilation Script"
echo "  Online Judge Sandbox Environment"
echo "=========================================="
echo ""
echo "Compiler: $COMPILER"
echo "Build command: $COMPILER $BUILD_CMD"
echo "Linker flags: $LDFLAGS (strip debug info)"
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
    echo "  - main.go (required) - Your Go source code"
    echo "  - Must have 'package main' declaration"
    echo "  - Must have 'func main()' function"
    echo ""
    exit $EXIT_FILE_ERROR
fi

# Display Go version
echo "Go version:"
$COMPILER version
echo ""

# Pre-compilation source check
echo "Pre-compilation checks..."
if ! check_go_source "$SOURCE_FILE"; then
    echo ""
    echo "=========================================="
    echo "  PRE-COMPILATION CHECK FAILED"
    echo "=========================================="
    echo ""
    echo "Fix the above errors before compilation"
    show_go_common_errors
    exit $EXIT_PACKAGE_ERROR
fi
echo ""

# Run compilation with timeout
echo "Compiling..."
COMPILER_OUTPUT=$(timeout $TIME_LIMIT $COMPILER $BUILD_CMD -o "$OUTPUT_FILE" $LDFLAGS "$SOURCE_FILE" 2>&1)
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
    echo "  - Debug info stripped (smaller binary)"
    echo "  - Time factor: 1.0 (native speed)"
    echo "  - Memory factor: 1.0 (standard memory)"
    echo ""

    show_go_performance_tips

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
    echo "  - Complex dependencies"
    echo "  - Slow module resolution"
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
    PARSED_ERROR=$(parse_go_error "$COMPILER_OUTPUT")
    if [ -n "$PARSED_ERROR" ]; then
        echo "Parsed error:"
        echo "$PARSED_ERROR"
    fi

    # Show common error help
    show_go_common_errors

    exit $EXIT_COMPILE_ERROR
fi