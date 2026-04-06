#!/bin/bash

# Python3 Execution Script for Online Judge
# Language: Python 3
# Time Factor: 2.0 (slower)
# Memory Factor: 1.5 (uses more memory)
# Note: Python does not require compilation

# Interpreter settings
INTERPRETER="python3"
EXTRA_FLAGS="-S"  # Skip site imports for faster startup

# Source file
SOURCE_FILE="main.py"

echo "=== Python3 Execution Script ==="
echo "Interpreter: $INTERPRETER"
echo "Source: $SOURCE_FILE"
echo ""
echo "Note: Python does not require compilation - the source is executed directly"
echo ""

# Check if source file exists
if [ ! -f "$SOURCE_FILE" ]; then
    echo "Error: Source file '$SOURCE_FILE' not found"
    exit 1
fi

# Display Python version
echo "Python version:"
$INTERPRETER --version
echo ""

# Syntax check (optional but helpful)
echo "Checking syntax..."
$INTERPRETER -m py_compile "$SOURCE_FILE" 2>&1

SYNTAX_EXIT_CODE=$?

if [ $SYNTAX_EXIT_CODE -eq 0 ]; then
    echo "Syntax check passed!"
    echo ""
    echo "To run the program:"
    echo "  $INTERPRETER $EXTRA_FLAGS $SOURCE_FILE"
    echo ""
    echo "Common Python runtime errors:"
    echo "  - SyntaxError: Check for syntax issues"
    echo "  - NameError/Undefined: Check for undefined variables"
    echo "  - TypeError: Check for type mismatches"
    echo "  - IndexError: Check for out-of-bounds array access"
    exit 0
else
    echo ""
    echo "Error: Syntax check failed"
    echo ""
    echo "Common Python syntax errors:"
    echo "  - Missing parentheses"
    echo "  - Incorrect indentation"
    echo "  - Missing colons after if/for/while/def"
    exit 1
fi