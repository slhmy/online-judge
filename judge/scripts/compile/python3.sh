#!/bin/bash

# Enhanced Python3 Execution Script for Online Judge
# Language: Python 3.11
# Time Factor: 2.0 (interpreter overhead)
# Memory Factor: 1.5 (GC and interpreter memory)
# Features: Optimized execution settings for competitive programming

set -e

# ==================== Configuration ====================
INTERPRETER="python3"
# -S: Skip site module initialization for faster startup
# -B: Don't write .pyc files (reduces disk I/O)
EXTRA_FLAGS="-S -B"

# Source file
SOURCE_FILE="main.py"

# Exit codes
EXIT_SUCCESS=0
EXIT_SYNTAX_ERROR=1
EXIT_RUNTIME_HINT=2
EXIT_FILE_ERROR=4

# ==================== Helper Functions ====================

parse_python_error() {
    local output="$1"
    local error_type=""
    local line_num=""
    local message=""

    # Parse SyntaxError
    if echo "$output" | grep -qE "SyntaxError:"; then
        error_type="SyntaxError"
        line_num=$(echo "$output" | grep -oE "line \d+" | grep -oE "\d+" | head -1)
        message=$(echo "$output" | grep -oE "SyntaxError: .*" | sed 's/SyntaxError: //')
    fi

    # Parse NameError
    if echo "$output" | grep -qE "NameError:"; then
        error_type="NameError"
        message=$(echo "$output" | grep -oE "NameError: .*" | sed 's/NameError: //')
    fi

    # Parse TypeError
    if echo "$output" | grep -qE "TypeError:"; then
        error_type="TypeError"
        message=$(echo "$output" | grep -oE "TypeError: .*" | sed 's/TypeError: //')
    fi

    # Parse IndexError
    if echo "$output" | grep -qE "IndexError:"; then
        error_type="IndexError"
        message=$(echo "$output" | grep -oE "IndexError: .*" | sed 's/IndexError: //')
    fi

    # Parse ValueError
    if echo "$output" | grep -qE "ValueError:"; then
        error_type="ValueError"
        message=$(echo "$output" | grep -oE "ValueError: .*" | sed 's/ValueError: //')
    fi

    if [ -n "$error_type" ]; then
        if [ -n "$line_num" ]; then
            echo "$error_type at line $line_num: $message"
        else
            echo "$error_type: $message"
        fi
    fi
}

show_python_common_errors() {
    echo ""
    echo "Common Python3 errors and solutions:"
    echo ""
    echo "  [Syntax Errors]"
    echo "    - Missing parentheses: Add () for function calls"
    echo "    - Invalid indentation: Use consistent spaces (4 recommended)"
    echo "    - Missing colon: Add ':' after if/for/while/def/class"
    echo "    - Unexpected EOF: Check for missing brackets or quotes"
    echo ""
    echo "  [Name Errors]"
    echo "    - Name not defined: Define variable before use or import module"
    echo "    - Cannot find reference: Check spelling and scope"
    echo ""
    echo "  [Type Errors]"
    echo "    - Unsupported operand: Check type compatibility"
    echo "    - Not iterable: Ensure object can be iterated"
    echo "    - Missing argument: Provide required function arguments"
    echo ""
    echo "  [Index Errors]"
    echo "    - List index out of range: Check array bounds"
    echo "    - Key not found: Verify dictionary key exists"
    echo ""
    echo "  [Value Errors]"
    echo "    - Invalid literal: Check conversion function input"
    echo "    - Too many values: Match unpacking count"
    echo ""
}

check_python_source() {
    local source="$1"
    local warnings=""

    # Check for potential issues
    if grep -qE "import sys" "$source" && ! grep -qE "sys.stdin|sys.stdout|input\(" "$source"; then
        warnings+="Note: sys imported but no sys I/O used\n"
    fi

    if grep -qE "input\(\)" "$source" && grep -qE "for.*range.*input\(\)"; then
        warnings+="Tip: For large input, use sys.stdin.read() for faster I/O\n"
    fi

    if grep -qE "print\(.*\+.*\)" "$source"; then
        warnings+="Tip: Use f-strings or format() for string concatenation\n"
    fi

    if grep -qE "range\(\d{6,}\)" "$source"; then
        warnings+="Warning: Very large range - may cause timeout\n"
    fi

    if grep -qE "sorted\(.*\)" "$source" && grep -qE "for.*in.*sorted"; then
        warnings+="Tip: Consider sort() for in-place sorting to save memory\n"
    fi

    # Check for recursion depth issues
    if grep -qE "def.*\(|def.*\)" "$source" && grep -qE "return.*\w+\("; then
        warnings+="Note: If using deep recursion, set sys.setrecursionlimit()\n"
    fi

    if [ -n "$warnings" ]; then
        echo "$warnings"
    fi
}

show_performance_tips() {
    echo ""
    echo "Python3 Performance Tips for Competitive Programming:"
    echo ""
    echo "  [Input Optimization]"
    echo "    import sys"
    echo "    data = sys.stdin.read().split()"
    echo "    # Much faster than multiple input() calls"
    echo ""
    echo "  [Output Optimization]"
    echo "    # Use sys.stdout.write() or print with end='' for bulk output"
    echo "    print('\\n'.join(results))  # Instead of multiple print() calls"
    echo ""
    echo "  [Data Structures]"
    echo "    - Use list for ordered sequences"
    echo "    - Use set for fast membership testing"
    echo "    - Use dict for key-value lookups"
    echo "    - Use deque from collections for queue operations"
    echo ""
    echo "  [Memory Optimization]"
    echo "    - Use generators instead of lists when possible"
    echo "    - Avoid deep copies - use references"
    echo "    - Clear large variables after use"
    echo ""
    echo "  [Recursion]"
    echo "    import sys"
    echo "    sys.setrecursionlimit(10**6)  # For deep recursion"
    echo ""
}

# ==================== Main Execution ====================

echo "=========================================="
echo "  Python3 Execution Script"
echo "  Online Judge Sandbox Environment"
echo "=========================================="
echo ""
echo "Interpreter: $INTERPRETER"
echo "Flags: $EXTRA_FLAGS"
echo "Source: $SOURCE_FILE"
echo ""
echo "Note: Python does not require compilation"
echo "      Source code is executed directly by the interpreter"
echo ""

# Check if source file exists
if [ ! -f "$SOURCE_FILE" ]; then
    echo "Error: Source file '$SOURCE_FILE' not found"
    echo ""
    echo "Expected file structure:"
    echo "  - main.py (required) - Your Python source code"
    echo ""
    exit $EXIT_FILE_ERROR
fi

# Display Python version
echo "Python version:"
$INTERPRETER --version
echo ""

# Pre-execution source check
echo "Pre-execution checks..."
check_python_source "$SOURCE_FILE"
echo ""

# Syntax check using py_compile module
echo "Checking syntax..."
SYNTAX_OUTPUT=$($INTERPRETER -m py_compile "$SOURCE_FILE" 2>&1)
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
    echo "  - Site module skipped for faster startup (-S)"
    echo "  - Bytecode files not written (-B)"
    echo "  - Time factor: 2.0 (interpreter overhead)"
    echo "  - Memory factor: 1.5 (interpreter overhead)"
    echo ""

    show_performance_tips

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
    PARSED_ERROR=$(parse_python_error "$SYNTAX_OUTPUT")
    if [ -n "$PARSED_ERROR" ]; then
        echo "Parsed error:"
        echo "$PARSED_ERROR"
    fi

    # Show common error help
    show_python_common_errors

    exit $EXIT_SYNTAX_ERROR
fi