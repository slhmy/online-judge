#!/bin/bash

# Enhanced Node.js 18 Execution Script for Online Judge
# Language: Node.js 18
# Time Factor: 2.0 (V8 overhead)
# Memory Factor: 1.5 (GC and runtime overhead)
# Features: Optimized execution settings for competitive programming

set -e

# ==================== Configuration ====================
INTERPRETER="node"
# --optimize_for_size: Reduce memory footprint
# --max-old-space-size=512: Limit heap to 512MB
EXTRA_FLAGS="--optimize_for_size --max-old-space-size=512"

# Source file
SOURCE_FILE="main.js"

# Exit codes
EXIT_SUCCESS=0
EXIT_SYNTAX_ERROR=1
EXIT_RUNTIME_HINT=2
EXIT_FILE_ERROR=4

# ==================== Helper Functions ====================

parse_nodejs_error() {
    local output="$1"
    local error_type=""
    local line_num=""
    local col_num=""
    local message=""

    # Parse SyntaxError
    if echo "$output" | grep -qE "SyntaxError:"; then
        error_type="SyntaxError"
        message=$(echo "$output" | grep -oE "SyntaxError: .*" | sed 's/SyntaxError: //')
        # Try to extract line number from stack trace
        if echo "$output" | grep -qE "at .+ \(main\.js:\d+:\d+\)"; then
            line_num=$(echo "$output" | grep -oE "main\.js:\d+" | grep -oE "\d+" | head -1)
        fi
    fi

    # Parse ReferenceError
    if echo "$output" | grep -qE "ReferenceError:"; then
        error_type="ReferenceError"
        message=$(echo "$output" | grep -oE "ReferenceError: .*" | sed 's/ReferenceError: //')
        if echo "$output" | grep -qE "at .+ \(main\.js:\d+:\d+\)"; then
            line_num=$(echo "$output" | grep -oE "main\.js:\d+" | grep -oE "\d+" | head -1)
        fi
    fi

    # Parse TypeError
    if echo "$output" | grep -qE "TypeError:"; then
        error_type="TypeError"
        message=$(echo "$output" | grep -oE "TypeError: .*" | sed 's/TypeError: //')
        if echo "$output" | grep -qE "at .+ \(main\.js:\d+:\d+\)"; then
            line_num=$(echo "$output" | grep -oE "main\.js:\d+" | grep -oE "\d+" | head -1)
        fi
    fi

    # Parse RangeError
    if echo "$output" | grep -qE "RangeError:"; then
        error_type="RangeError"
        message=$(echo "$output" | grep -oE "RangeError: .*" | sed 's/RangeError: //')
    fi

    if [ -n "$error_type" ]; then
        if [ -n "$line_num" ]; then
            echo "$error_type at line $line_num: $message"
        else
            echo "$error_type: $message"
        fi
    fi
}

show_nodejs_common_errors() {
    echo ""
    echo "Common JavaScript/Node.js errors and solutions:"
    echo ""
    echo "  [Syntax Errors]"
    echo "    - Missing bracket/parenthesis: Balance {} [] ()"
    echo "    - Unexpected token: Check for typos"
    echo "    - Missing semicolon (rare): Add ';' if needed"
    echo "    - Invalid variable name: Use valid identifiers"
    echo ""
    echo "  [Reference Errors]"
    echo "    - Variable not defined: Declare before use"
    echo "    - Function not found: Define or import function"
    echo "    - Cannot access before initialization: Check let/const scope"
    echo ""
    echo "  [Type Errors]"
    echo "    - Cannot read property: Check object exists"
    echo "    - Undefined is not a function: Verify callable"
    echo "    - Assignment to constant: Use let instead of const"
    echo ""
    echo "  [Range Errors]"
    echo "    - Invalid array length: Use valid positive integer"
    echo "    - Out of range index: Check bounds before access"
    echo ""
}

check_nodejs_source() {
    local source="$1"
    local warnings=""

    # Check for potential issues
    if grep -qE "require\(.*\)" "$source" && ! grep -qE "fs|readline|process"; then
        warnings+="Note: Common modules not imported\n"
    fi

    # Check for async/await usage
    if grep -qE "await" "$source" && ! grep -qE "async"; then
        warnings+="Warning: await used without async function\n"
    fi

    # Check for console.log usage (performance hint)
    if grep -qE "console\.log" "$source"; then
        warnings+="Tip: Use process.stdout.write for bulk output\n"
    fi

    # Check for potential blocking issues
    if grep -qE "readFileSync|writeFileSync" "$source"; then
        warnings+="Note: Sync I/O operations used (OK for CP)\n"
    fi

    # Check for memory-intensive patterns
    if grep -qE "\.map\(.*\)\.filter\|\.filter\(.*\)\.map"; then
        warnings+="Tip: Consider single loop for combined map/filter\n"
    fi

    if [ -n "$warnings" ]; then
        echo "$warnings"
    fi
}

show_nodejs_performance_tips() {
    echo ""
    echo "Node.js 18 Performance Tips for Competitive Programming:"
    echo ""
    echo "  [Input Optimization]"
    echo "    const fs = require('fs');"
    echo "    const data = fs.readFileSync(0, 'utf-8').split(/\s+/);"
    echo "    // Much faster than readline"
    echo ""
    echo "  [Output Optimization]"
    echo "    // Use process.stdout.write for bulk output"
    echo "    process.stdout.write(result + '\\n');"
    echo "    // Or build array and write once"
    echo "    process.stdout.write(results.join('\\n'));"
    echo ""
    echo "  [Data Structures]"
    echo "    - Array: Dynamic array, push/pop"
    echo "    - Object: Hash table (use for key-value)"
    echo "    - Map: Better for non-string keys"
    echo "    - Set: Fast membership testing"
    echo ""
    echo "  [Memory Optimization]"
    echo "    - Avoid deep copy: Use references"
    echo "    - Clear large arrays: arr.length = 0"
    echo "    - Use typed arrays for numeric data"
    echo ""
    echo "  [Async Handling]"
    echo "    // For competitive programming, avoid async"
    echo "    // Use synchronous I/O for simplicity"
    echo "    const input = fs.readFileSync(0, 'utf-8');"
    echo ""
    echo "  [Standard Template]"
    echo "    const fs = require('fs');"
    echo "    const input = fs.readFileSync(0, 'utf-8').trim();"
    echo "    const lines = input.split('\\n');"
    echo "    // Your solution here"
    echo "    process.stdout.write(result + '\\n');"
    echo ""
}

# ==================== Main Execution ====================

echo "=========================================="
echo "  Node.js 18 Execution Script"
echo "  Online Judge Sandbox Environment"
echo "=========================================="
echo ""
echo "Interpreter: $INTERPRETER"
echo "Flags: $EXTRA_FLAGS"
echo "Source: $SOURCE_FILE"
echo ""
echo "Note: JavaScript does not require compilation"
echo "      Source code is executed directly by the V8 engine"
echo ""

# Check if source file exists
if [ ! -f "$SOURCE_FILE" ]; then
    echo "Error: Source file '$SOURCE_FILE' not found"
    echo ""
    echo "Expected file structure:"
    echo "  - main.js (required) - Your JavaScript source code"
    echo ""
    exit $EXIT_FILE_ERROR
fi

# Display Node.js version
echo "Node.js version:"
$INTERPRETER --version
echo "V8 version:"
$INTERPRETER -e "console.log(process.versions.v8)"
echo ""

# Pre-execution source check
echo "Pre-execution checks..."
check_nodejs_source "$SOURCE_FILE"
echo ""

# Syntax check using Node.js --check flag
echo "Checking syntax..."
SYNTAX_OUTPUT=$($INTERPRETER --check "$SOURCE_FILE" 2>&1)
SYNTAX_EXIT_CODE=$?

if [ $SYNTAX_EXIT_CODE -eq 0 ]; then
    echo "=========================================="
    echo "  SYNTAX CHECK PASSED"
    echo "=========================================="
    echo ""
    echo "To run the program:"
    echo "  $INTERPRETER $EXTRA_FLAGS $SOURCE_FILE"
    echo ""
    echo "Execution notes:"
    echo "  - Heap size limited to 512MB"
    echo "  - Optimized for size (--optimize_for_size)"
    echo "  - Time factor: 2.0 (V8 overhead)"
    echo "  - Memory factor: 1.5 (runtime overhead)"
    echo ""

    show_nodejs_performance_tips

    exit $EXIT_SUCCESS

else
    echo ""
    echo "=========================================="
    echo "  SYNTAX CHECK FAILED"
    echo "=========================================="
    echo ""
    echo "Compiler output:"
    echo "$SYNTAX_OUTPUT"
    echo ""

    # Parse and show structured error
    PARSED_ERROR=$(parse_nodejs_error "$SYNTAX_OUTPUT")
    if [ -n "$PARSED_ERROR" ]; then
        echo "Parsed error:"
        echo "$PARSED_ERROR"
    fi

    # Show common error help
    show_nodejs_common_errors

    exit $EXIT_SYNTAX_ERROR
fi