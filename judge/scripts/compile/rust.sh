#!/bin/bash

# Enhanced Rust Compilation Script for Online Judge
# Language: Rust (stable channel)
# Time Factor: 1.0 (native speed)
# Memory Factor: 1.0 (standard memory)
# Features: Optimized for competitive programming

set -e

# ==================== Configuration ====================
COMPILER="rustc"
OPTIMIZATION="-O"  # Release mode optimization

# Alternative: Use cargo for complex projects
CARGO="cargo"
CARGO_BUILD="build --release"

# Source and output files
SOURCE_FILE="main.rs"
OUTPUT_FILE="main"

# Compilation resource limits (Rust compilation can be slow)
TIME_LIMIT=120       # seconds
MEMORY_LIMIT_MB=1024 # megabytes

# Exit codes
EXIT_SUCCESS=0
EXIT_COMPILE_ERROR=1
EXIT_TIMEOUT=2
EXIT_MAIN_ERROR=3
EXIT_FILE_ERROR=4

# ==================== Helper Functions ====================

parse_rust_error() {
    local output="$1"
    local error_type=""
    local line_num=""
    local col_num=""
    local message=""
    local error_code=""

    # Parse error code (E0XXX)
    if echo "$output" | grep -qE "error\[E\d+\]"; then
        error_code=$(echo "$output" | grep -oE "E\d+" | head -1)
    fi

    # Parse structured Rust errors with location
    if echo "$output" | grep -qE "--> main\.rs:\d+:\d+"; then
        line_num=$(echo "$output" | grep -oE "main\.rs:\d+" | grep -oE "\d+" | head -1)
        col_num=$(echo "$output" | grep -oE ":\d+" | tail -1 | grep -oE "\d+")
    fi

    # Parse error message
    if echo "$output" | grep -qE "error(?:\[E\d+\])?: "; then
        message=$(echo "$output" | grep -oE "error(?:\[E\d+\])?: .*" | sed 's/error(\[E\d+\])?: //' | head -1)
    fi

    # Handle specific error types
    if echo "$output" | grep -qE "cannot find"; then
        message="Cannot find value/function in scope"
    fi

    if echo "$output" | grep -qE "mismatched types"; then
        message="Mismatched types"
    fi

    if [ -n "$message" ]; then
        if [ -n "$line_num" ] && [ -n "$col_num" ]; then
            echo "error at line $line_num:$col_num: $message"
            if [ -n "$error_code" ]; then
                echo "  (Error code: $error_code)"
            fi
        elif [ -n "$line_num" ]; then
            echo "error at line $line_num: $message"
        else
            echo "error: $message"
            if [ -n "$error_code" ]; then
                echo "  (Error code: $error_code)"
            fi
        fi
    fi
}

show_rust_common_errors() {
    echo ""
    echo "Common Rust compilation errors and solutions:"
    echo ""
    echo "  [Scope Errors (E0425)]"
    echo "    - Cannot find value/function: Define or import it"
    echo "    - Use of undeclared: Add declaration"
    echo ""
    echo "  [Type Errors (E0308)]"
    echo "    - Mismatched types: Use type conversion"
    echo "    - Expected X, found Y: Check type annotations"
    echo "    - Cannot convert: Use From/Into trait"
    echo ""
    echo "  [Method Errors (E0599)]"
    echo "    - No method named X: Implement trait or use correct type"
    echo "    - Method not found: Check trait imports"
    echo ""
    echo "  [Ownership Errors]"
    echo "    - Use of moved value: Use Clone or reference"
    echo "    - Borrow after move: Check ownership semantics"
    echo "    - Cannot borrow as mutable: Use &mut appropriately"
    echo ""
    echo "  [Lifetime Errors (E0597)]"
    echo "    - Borrowed value does not live long enough: Check lifetime"
    echo "    - Lifetime may not be long enough: Add lifetime annotation"
    echo ""
    echo "  [Syntax Errors]"
    echo "    - Expected identifier: Check syntax"
    echo "    - Unexpected token: Fix syntax issues"
    echo ""
}

check_rust_source() {
    local source="$1"
    local warnings=""
    local errors=""

    # Check for main function (required)
    if ! grep -qE "fn main\("; then
        errors+="Error: Missing main function\n"
    fi

    # Check for common issues
    if grep -qE "println!" "$source" && ! grep -qE "use std::"; then
        warnings+="Note: println! is in prelude, no import needed\n"
    fi

    # Check for potential performance issues
    if grep -qE "Vec::new()" "$source" && grep -qE "push"; then
        warnings+="Tip: Use Vec::with_capacity(n) when size is known\n"
    fi

    if grep -qE "String::from" "$source"; then
        warnings+="Tip: Use .to_string() or format! for string creation\n"
    fi

    # Check for unsafe code (not typically needed in CP)
    if grep -qE "unsafe" "$source"; then
        warnings+="Warning: unsafe block detected - may not be needed\n"
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

show_rust_performance_tips() {
    echo ""
    echo "Rust Performance Tips for Competitive Programming:"
    echo ""
    echo "  [Input Optimization]"
    echo "    use std::io::{self, BufRead};"
    echo "    let stdin = io::stdin();"
    echo "    let reader = stdin.lock();"
    echo "    for line in reader.lines() { ... }"
    echo ""
    echo "  [Output Optimization]"
    echo "    use std::io::Write;"
    echo "    let stdout = io::stdout();"
    echo "    let mut out = stdout.lock();"
    echo "    writeln!(out, \"result: {}\", x).unwrap();"
    echo ""
    echo "  [Data Structures]"
    echo "    - Vec<T>: Dynamic array"
    echo "    - HashMap<K,V>: O(1) average lookup"
    echo "    - BTreeMap<K,V>: Sorted map, O(log n)"
    echo "    - VecDeque<T>: Double-ended queue"
    echo ""
    echo "  [Memory Optimization]"
    echo "    - Preallocate: Vec::with_capacity(n)"
    echo "    - Use iterators instead of index loops"
    echo "    - Consider Box<T> for large data"
    echo ""
    echo "  [Standard Template]"
    echo "    fn main() {"
    echo "        use std::io::{self, BufRead, Write};"
    echo "        let stdin = io::stdin();"
    echo "        let reader = stdin.lock();"
    echo "        let stdout = io::stdout();"
    echo "        let mut out = stdout.lock();"
    echo "        // Your solution here"
    echo "    }"
    echo ""
}

# ==================== Main Compilation ====================

echo "=========================================="
echo "  Rust Compilation Script"
echo "  Online Judge Sandbox Environment"
echo "=========================================="
echo ""
echo "Compiler: $COMPILER"
echo "Optimization: $OPTIMIZATION (release mode)"
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
    echo "  - main.rs (required) - Your Rust source code"
    echo "  - Must have 'fn main()' function"
    echo ""
    exit $EXIT_FILE_ERROR
fi

# Display Rust version
echo "Rust version:"
$COMPILER --version
cargo --version 2>/dev/null || true
echo ""

# Pre-compilation source check
echo "Pre-compilation checks..."
if ! check_rust_source "$SOURCE_FILE"; then
    echo ""
    echo "=========================================="
    echo "  PRE-COMPILATION CHECK FAILED"
    echo "=========================================="
    echo ""
    echo "Fix the above errors before compilation"
    show_rust_common_errors
    exit $EXIT_MAIN_ERROR
fi
echo ""

# Run compilation with timeout
echo "Compiling..."
COMPILER_OUTPUT=$(timeout $TIME_LIMIT $COMPILER $OPTIMIZATION -o "$OUTPUT_FILE" "$SOURCE_FILE" 2>&1)
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
    echo "  - Optimized with -O (release mode)"
    echo "  - Time factor: 1.0 (native speed)"
    echo "  - Memory factor: 1.0 (standard memory)"
    echo ""

    show_rust_performance_tips

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
    echo "  - Complex type inference"
    echo "  - Large monomorphization"
    echo "  - Heavy optimization workload"
    echo ""
    echo "Tip: Try using cargo with incremental compilation"
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
    PARSED_ERROR=$(parse_rust_error "$COMPILER_OUTPUT")
    if [ -n "$PARSED_ERROR" ]; then
        echo "Parsed error:"
        echo "$PARSED_ERROR"
    fi

    # Show common error help
    show_rust_common_errors

    exit $EXIT_COMPILE_ERROR
fi