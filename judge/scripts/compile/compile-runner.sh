#!/bin/bash

# Universal Compile Runner Script for Online Judge
# This script provides a unified interface for compiling/checking all supported languages
# Usage: compile-runner.sh <language> [options]
# Languages: cpp, c, python3, java, go, rust, nodejs

set -e

# ==================== Configuration ====================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Exit codes
EXIT_SUCCESS=0
EXIT_COMPILE_ERROR=1
EXIT_TIMEOUT=2
EXIT_UNSUPPORTED=3
EXIT_FILE_ERROR=4

# Language configuration
declare -A LANG_CONFIG

# C++17
LANG_CONFIG["cpp_compiler"]="g++"
LANG_CONFIG["cpp_standard"]="-std=c++17"
LANG_CONFIG["cpp_flags"]="-O2 -lm -DONLINE_JUDGE -pipe -fno-stack-limit"
LANG_CONFIG["cpp_source"]="main.cpp"
LANG_CONFIG["cpp_output"]="main"
LANG_CONFIG["cpp_timeout"]=30
LANG_CONFIG["cpp_needs_compile"]=true

# C11
LANG_CONFIG["c_compiler"]="gcc"
LANG_CONFIG["c_standard"]="-std=c11"
LANG_CONFIG["c_flags"]="-O2 -lm -DONLINE_JUDGE -pipe"
LANG_CONFIG["c_source"]="main.c"
LANG_CONFIG["c_output"]="main"
LANG_CONFIG["c_timeout"]=30
LANG_CONFIG["c_needs_compile"]=true

# Python3
LANG_CONFIG["python3_interpreter"]="python3"
LANG_CONFIG["python3_flags"]="-S -B"
LANG_CONFIG["python3_source"]="main.py"
LANG_CONFIG["python3_needs_compile"]=false
LANG_CONFIG["python3_timeout"]=5

# Java 17
LANG_CONFIG["java_compiler"]="javac"
LANG_CONFIG["java_flags"]="-encoding UTF8 -source 17 -target 17 -Xlint:all"
LANG_CONFIG["java_source"]="Main.java"
LANG_CONFIG["java_class"]="Main.class"
LANG_CONFIG["java_timeout"]=60
LANG_CONFIG["java_needs_compile"]=true
LANG_CONFIG["java_run"]="java -Xmx512M -Xss64M -DONLINE_JUDGE=true -enableassertions Main"

# Go 1.21
LANG_CONFIG["go_compiler"]="go"
LANG_CONFIG["go_cmd"]="build"
LANG_CONFIG["go_flags"]="-ldflags -s -w"
LANG_CONFIG["go_source"]="main.go"
LANG_CONFIG["go_output"]="main"
LANG_CONFIG["go_timeout"]=60
LANG_CONFIG["go_needs_compile"]=true

# Rust
LANG_CONFIG["rust_compiler"]="rustc"
LANG_CONFIG["rust_flags"]="-O"
LANG_CONFIG["rust_source"]="main.rs"
LANG_CONFIG["rust_output"]="main"
LANG_CONFIG["rust_timeout"]=120
LANG_CONFIG["rust_needs_compile"]=true

# Node.js 18
LANG_CONFIG["nodejs_interpreter"]="node"
LANG_CONFIG["nodejs_flags"]="--optimize_for_size --max-old-space-size=512"
LANG_CONFIG["nodejs_source"]="main.js"
LANG_CONFIG["nodejs_needs_compile"]=false
LANG_CONFIG["nodejs_timeout"]=5

# ==================== Helper Functions ====================

show_usage() {
    echo "Universal Compile Runner for Online Judge"
    echo ""
    echo "Usage: $0 <language> [--source <file>] [--output <file>] [--timeout <seconds>]"
    echo ""
    echo "Supported languages:"
    echo "  cpp       - C++17 (g++)"
    echo "  c         - C11 (gcc)"
    echo "  python3   - Python 3.11"
    echo "  java      - Java 17 (OpenJDK)"
    echo "  go        - Go 1.21"
    echo "  rust      - Rust (stable)"
    echo "  nodejs    - Node.js 18"
    echo ""
    echo "Options:"
    echo "  --source <file>    Source file path (default: language-specific)"
    echo "  --output <file>    Output file path (default: language-specific)"
    echo "  --timeout <sec>    Compilation timeout (default: language-specific)"
    echo "  --check-only       Only check syntax, don't compile"
    echo "  --verbose          Show detailed output"
    echo ""
    echo "Examples:"
    echo "  $0 cpp                           # Compile main.cpp to main"
    echo "  $0 python3 --check-only          # Syntax check main.py"
    echo "  $0 java --source MyMain.java     # Compile MyMain.java"
    echo ""
}

show_language_info() {
    echo ""
    echo "Supported Languages and Configurations:"
    echo ""
    echo "  Language    Version      Time Factor  Memory Factor  Compile?"
    echo "  --------------------------------------------------------------"
    echo "  cpp         C++17 (g++)  1.0          1.0            Yes"
    echo "  c           C11 (gcc)    1.0          1.0            Yes"
    echo "  python3     Python 3.11  2.0          1.5            No (interpreted)"
    echo "  java        Java 17      2.0          1.5            Yes"
    echo "  go          Go 1.21      1.0          1.0            Yes"
    echo "  rust        Rust stable  1.0          1.0            Yes"
    echo "  nodejs      Node.js 18   2.0          1.5            No (interpreted)"
    echo ""
}

validate_language() {
    local lang="$1"
    case "$lang" in
        cpp|c|python3|java|go|rust|nodejs)
            return 0
            ;;
        *)
            echo "Error: Unsupported language '$lang'"
            show_language_info
            return 1
            ;;
    esac
}

# ==================== Language-Specific Compilers ====================

compile_cpp() {
    local source="${LANG_CONFIG[cpp_source]}"
    local output="${LANG_CONFIG[cpp_output]}"
    local timeout="${LANG_CONFIG[cpp_timeout]}"
    local check_only=false
    local verbose=false

    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --source) source="$2"; shift 2 ;;
            --output) output="$2"; shift 2 ;;
            --timeout) timeout="$2"; shift 2 ;;
            --check-only) check_only=true; shift ;;
            --verbose) verbose=true; shift ;;
            *) shift ;;
        esac
    done

    if [ ! -f "$source" ]; then
        echo "Error: Source file '$source' not found"
        return $EXIT_FILE_ERROR
    fi

    if $verbose; then
        echo "Compiling C++17..."
        echo "  Source: $source"
        echo "  Output: $output"
        echo "  Timeout: $timeout seconds"
    fi

    local compiler="${LANG_CONFIG[cpp_compiler]}"
    local standard="${LANG_CONFIG[cpp_standard]}"
    local flags="${LANG_CONFIG[cpp_flags]}"

    if $check_only; then
        # Syntax check only (dry run)
        $compiler $standard $flags -fsyntax-only "$source" 2>&1
        return $?
    fi

    local result
    result=$(timeout $timeout $compiler $standard $flags -o "$output" "$source" 2>&1)
    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        echo "Compilation successful"
        echo "Output: $output"
        return $EXIT_SUCCESS
    elif [ $exit_code -eq 124 ]; then
        echo "Compilation timeout after $timeout seconds"
        return $EXIT_TIMEOUT
    else
        echo "Compilation failed"
        echo "$result"
        return $EXIT_COMPILE_ERROR
    fi
}

compile_c() {
    local source="${LANG_CONFIG[c_source]}"
    local output="${LANG_CONFIG[c_output]}"
    local timeout="${LANG_CONFIG[c_timeout]}"
    local check_only=false
    local verbose=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --source) source="$2"; shift 2 ;;
            --output) output="$2"; shift 2 ;;
            --timeout) timeout="$2"; shift 2 ;;
            --check-only) check_only=true; shift ;;
            --verbose) verbose=true; shift ;;
            *) shift ;;
        esac
    done

    if [ ! -f "$source" ]; then
        echo "Error: Source file '$source' not found"
        return $EXIT_FILE_ERROR
    fi

    if $verbose; then
        echo "Compiling C11..."
        echo "  Source: $source"
        echo "  Output: $output"
    fi

    local compiler="${LANG_CONFIG[c_compiler]}"
    local standard="${LANG_CONFIG[c_standard]}"
    local flags="${LANG_CONFIG[c_flags]}"

    if $check_only; then
        $compiler $standard $flags -fsyntax-only "$source" 2>&1
        return $?
    fi

    local result
    result=$(timeout $timeout $compiler $standard $flags -o "$output" "$source" 2>&1)
    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        echo "Compilation successful"
        echo "Output: $output"
        return $EXIT_SUCCESS
    elif [ $exit_code -eq 124 ]; then
        echo "Compilation timeout"
        return $EXIT_TIMEOUT
    else
        echo "Compilation failed"
        echo "$result"
        return $EXIT_COMPILE_ERROR
    fi
}

compile_python3() {
    local source="${LANG_CONFIG[python3_source]}"
    local timeout="${LANG_CONFIG[python3_timeout]}"
    local check_only=true  # Always just check syntax
    local verbose=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --source) source="$2"; shift 2 ;;
            --verbose) verbose=true; shift ;;
            *) shift ;;
        esac
    done

    if [ ! -f "$source" ]; then
        echo "Error: Source file '$source' not found"
        return $EXIT_FILE_ERROR
    fi

    if $verbose; then
        echo "Checking Python3 syntax..."
        echo "  Source: $source"
    fi

    local interpreter="${LANG_CONFIG[python3_interpreter]}"
    local result
    result=$(timeout $timeout $interpreter -m py_compile "$source" 2>&1)
    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        echo "Syntax check passed"
        echo "Run: $interpreter ${LANG_CONFIG[python3_flags]} $source"
        return $EXIT_SUCCESS
    else
        echo "Syntax check failed"
        echo "$result"
        return $EXIT_COMPILE_ERROR
    fi
}

compile_java() {
    local source="${LANG_CONFIG[java_source]}"
    local timeout="${LANG_CONFIG[java_timeout]}"
    local check_only=false
    local verbose=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --source) source="$2"; shift 2 ;;
            --timeout) timeout="$2"; shift 2 ;;
            --check-only) check_only=true; shift ;;
            --verbose) verbose=true; shift ;;
            *) shift ;;
        esac
    done

    if [ ! -f "$source" ]; then
        echo "Error: Source file '$source' not found"
        return $EXIT_FILE_ERROR
    fi

    # Validate class name
    local class_name=$(basename "$source" .java)
    if [ "$class_name" != "Main" ]; then
        echo "Warning: Java class should be named 'Main'"
        echo "  Found: $class_name"
    fi

    if $verbose; then
        echo "Compiling Java 17..."
        echo "  Source: $source"
    fi

    local compiler="${LANG_CONFIG[java_compiler]}"
    local flags="${LANG_CONFIG[java_flags]}"

    if $check_only; then
        # Java doesn't have a simple syntax check, so we compile without output
        $compiler $flags -d /tmp "$source" 2>&1
        return $?
    fi

    local result
    result=$(timeout $timeout $compiler $flags "$source" 2>&1)
    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        echo "Compilation successful"
        echo "Class: ${LANG_CONFIG[java_class]}"
        echo "Run: ${LANG_CONFIG[java_run]}"
        return $EXIT_SUCCESS
    elif [ $exit_code -eq 124 ]; then
        echo "Compilation timeout"
        return $EXIT_TIMEOUT
    else
        echo "Compilation failed"
        echo "$result"
        return $EXIT_COMPILE_ERROR
    fi
}

compile_go() {
    local source="${LANG_CONFIG[go_source]}"
    local output="${LANG_CONFIG[go_output]}"
    local timeout="${LANG_CONFIG[go_timeout]}"
    local check_only=false
    local verbose=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --source) source="$2"; shift 2 ;;
            --output) output="$2"; shift 2 ;;
            --timeout) timeout="$2"; shift 2 ;;
            --check-only) check_only=true; shift ;;
            --verbose) verbose=true; shift ;;
            *) shift ;;
        esac
    done

    if [ ! -f "$source" ]; then
        echo "Error: Source file '$source' not found"
        return $EXIT_FILE_ERROR
    fi

    if $verbose; then
        echo "Compiling Go 1.21..."
        echo "  Source: $source"
        echo "  Output: $output"
    fi

    local compiler="${LANG_CONFIG[go_compiler]}"
    local cmd="${LANG_CONFIG[go_cmd]}"
    local flags="${LANG_CONFIG[go_flags]}"

    if $check_only; then
        $compiler vet "$source" 2>&1
        return $?
    fi

    local result
    result=$(timeout $timeout $compiler $cmd -o "$output" $flags "$source" 2>&1)
    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        echo "Compilation successful"
        echo "Output: $output"
        return $EXIT_SUCCESS
    elif [ $exit_code -eq 124 ]; then
        echo "Compilation timeout"
        return $EXIT_TIMEOUT
    else
        echo "Compilation failed"
        echo "$result"
        return $EXIT_COMPILE_ERROR
    fi
}

compile_rust() {
    local source="${LANG_CONFIG[rust_source]}"
    local output="${LANG_CONFIG[rust_output]}"
    local timeout="${LANG_CONFIG[rust_timeout]}"
    local check_only=false
    local verbose=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --source) source="$2"; shift 2 ;;
            --output) output="$2"; shift 2 ;;
            --timeout) timeout="$2"; shift 2 ;;
            --check-only) check_only=true; shift ;;
            --verbose) verbose=true; shift ;;
            *) shift ;;
        esac
    done

    if [ ! -f "$source" ]; then
        echo "Error: Source file '$source' not found"
        return $EXIT_FILE_ERROR
    fi

    if $verbose; then
        echo "Compiling Rust..."
        echo "  Source: $source"
        echo "  Output: $output"
    fi

    local compiler="${LANG_CONFIG[rust_compiler]}"
    local flags="${LANG_CONFIG[rust_flags]}"

    if $check_only; then
        $compiler --emit=metadata -o /tmp/check "$source" 2>&1
        return $?
    fi

    local result
    result=$(timeout $timeout $compiler $flags -o "$output" "$source" 2>&1)
    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        echo "Compilation successful"
        echo "Output: $output"
        return $EXIT_SUCCESS
    elif [ $exit_code -eq 124 ]; then
        echo "Compilation timeout"
        return $EXIT_TIMEOUT
    else
        echo "Compilation failed"
        echo "$result"
        return $EXIT_COMPILE_ERROR
    fi
}

compile_nodejs() {
    local source="${LANG_CONFIG[nodejs_source]}"
    local timeout="${LANG_CONFIG[nodejs_timeout]}"
    local verbose=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --source) source="$2"; shift 2 ;;
            --verbose) verbose=true; shift ;;
            *) shift ;;
        esac
    done

    if [ ! -f "$source" ]; then
        echo "Error: Source file '$source' not found"
        return $EXIT_FILE_ERROR
    fi

    if $verbose; then
        echo "Checking Node.js syntax..."
        echo "  Source: $source"
    fi

    local interpreter="${LANG_CONFIG[nodejs_interpreter]}"
    local result
    result=$(timeout $timeout $interpreter --check "$source" 2>&1)
    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        echo "Syntax check passed"
        echo "Run: $interpreter ${LANG_CONFIG[nodejs_flags]} $source"
        return $EXIT_SUCCESS
    else
        echo "Syntax check failed"
        echo "$result"
        return $EXIT_COMPILE_ERROR
    fi
}

# ==================== Main Entry Point ====================

main() {
    if [[ $# -lt 1 ]]; then
        show_usage
        return 1
    fi

    local language="$1"
    shift

    case "$language" in
        --help|-h|help)
            show_usage
            return 0
            ;;
        --info|info)
            show_language_info
            return 0
            ;;
    esac

    validate_language "$language" || return $EXIT_UNSUPPORTED

    case "$language" in
        cpp)    compile_cpp "$@" ;;
        c)      compile_c "$@" ;;
        python3) compile_python3 "$@" ;;
        java)   compile_java "$@" ;;
        go)     compile_go "$@" ;;
        rust)   compile_rust "$@" ;;
        nodejs) compile_nodejs "$@" ;;
    esac
}

main "$@"