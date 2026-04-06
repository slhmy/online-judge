#!/bin/bash

# Node.js 18 Execution Script for Online Judge
# Language: Node.js 18
# Time Factor: 2.0 (overhead)
# Memory Factor: 1.5
# Note: JavaScript does not require compilation

# Interpreter settings
INTERPRETER="node"
EXTRA_FLAGS="--optimize_for_size"

# Source file
SOURCE_FILE="main.js"

echo "=== Node.js 18 Execution Script ==="
echo "Interpreter: $INTERPRETER"
echo "Source: $SOURCE_FILE"
echo ""
echo "Note: JavaScript does not require compilation - the source is executed directly"
echo ""

# Check if source file exists
if [ ! -f "$SOURCE_FILE" ]; then
    echo "Error: Source file '$SOURCE_FILE' not found"
    exit 1
fi

# Display Node.js version
echo "Node.js version:"
$INTERPRETER --version
echo ""

# Syntax check using Node.js
echo "Checking syntax..."
$INTERPRETER --check "$SOURCE_FILE" 2>&1

SYNTAX_EXIT_CODE=$?

if [ $SYNTAX_EXIT_CODE -eq 0 ]; then
    echo "Syntax check passed!"
    echo ""
    echo "To run the program:"
    echo "  $INTERPRETER $EXTRA_FLAGS $SOURCE_FILE"
    echo ""
    echo "Common JavaScript runtime errors:"
    echo "  - SyntaxError: Check for syntax issues"
    echo "  - ReferenceError: Variable/function not defined"
    echo "  - TypeError: Type-related issues"
    echo "  - RangeError: Value out of expected range"
    echo ""
    echo "Performance tips:"
    echo "  - Avoid blocking the event loop"
    echo "  - Use efficient data structures"
    echo "  - Minimize synchronous operations"
    exit 0
else
    echo ""
    echo "Error: Syntax check failed"
    echo ""
    echo "Common JavaScript syntax errors:"
    echo "  - Missing brackets or parentheses"
    echo "  - Unexpected tokens"
    echo "  - Invalid variable names"
    exit 1
fi