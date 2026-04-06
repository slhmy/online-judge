#!/bin/bash

# Build script for Judge Runtime Docker Image
# This creates the judge-runtime:latest image with all compilers

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGE_NAME="judge-runtime"
IMAGE_TAG="latest"

echo "=========================================="
echo "Building Judge Runtime Docker Image"
echo "=========================================="

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed or not available in PATH"
    exit 1
fi

# Build the Dockerfile in the judge directory
DOCKERFILE_PATH="${SCRIPT_DIR}/Dockerfile"

if [ ! -f "$DOCKERFILE_PATH" ]; then
    echo "Error: Dockerfile not found at $DOCKERFILE_PATH"
    exit 1
fi

echo ""
echo "Building image: ${IMAGE_NAME}:${IMAGE_TAG}"
echo "Using Dockerfile: $DOCKERFILE_PATH"
echo ""

# Build the image
docker build \
    -t "${IMAGE_NAME}:${IMAGE_TAG}" \
    -f "$DOCKERFILE_PATH" \
    "${SCRIPT_DIR}"

echo ""
echo "=========================================="
echo "Build completed successfully!"
echo "=========================================="
echo ""
echo "Verifying installed compilers..."

# Run health check to verify all compilers are installed
docker run --rm "${IMAGE_NAME}:${IMAGE_TAG}" /usr/local/bin/check-compilers

echo ""
echo "Image is ready for use: ${IMAGE_NAME}:${IMAGE_TAG}"
echo ""
echo "To test compilation for a specific language:"
echo "  docker run --rm -v /tmp/test:/workspace -w /workspace ${IMAGE_NAME}:${IMAGE_TAG} <compiler> <args>"
echo ""
echo "Examples:"
echo "  # C++ compilation"
echo "  docker run --rm -v /tmp/test:/workspace -w /workspace ${IMAGE_NAME}:${IMAGE_TAG} g++ -std=c++17 -O2 -o main main.cpp"
echo ""
echo "  # Java compilation"
echo "  docker run --rm -v /tmp/test:/workspace -w /workspace ${IMAGE_NAME}:${IMAGE_TAG} javac Main.java"
echo ""
echo "  # Go compilation"
echo "  docker run --rm -v /tmp/test:/workspace -w /workspace ${IMAGE_NAME}:${IMAGE_TAG} go build -o main main.go"
echo ""