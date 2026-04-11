# Judge Daemon Dockerfile - Multi-language compilation support
# This image contains all compilers for supported languages:
# - C++17 (g++)
# - Python3
# - Java17
# - Go 1.21
# - Rust
# - Node.js 18

# Build stage for Go daemon
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

# Copy gen module (pre-generated protobuf code)
COPY gen/ ./gen/

# Copy judge module and fix replace directive for Docker context
COPY judge/go.mod judge/go.sum ./judge/
WORKDIR /app/judge
RUN sed -i 's|=> ../gen|=> /app/gen|g' go.mod
RUN go mod download

# Copy judge source
WORKDIR /app
COPY judge/ ./judge/

WORKDIR /app/judge
RUN CGO_ENABLED=0 GOOS=linux go build -o /judgedaemon ./cmd/judgedaemon

# Runtime stage with all compilers
FROM ubuntu:22.04

# Avoid interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install base dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    curl \
    wget \
    gnupg \
    software-properties-common \
    docker-cli \
    && rm -rf /var/lib/apt/lists/*

# ==================== C++17 (g++) ====================
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    g++ \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Verify g++ version (should support C++17)
RUN g++ --version && echo "C++17 compiler installed"

# ==================== Python3 ====================
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    python3-pip \
    python3-dev \
    && rm -rf /var/lib/apt/lists/*

# Install common Python packages for competitive programming
RUN pip3 install --no-cache-dir \
    numpy \
    scipy \
    sympy \
    pypy3 || true

# Verify Python3 version
RUN python3 --version

# ==================== Java 17 ====================
RUN apt-get update && apt-get install -y --no-install-recommends \
    openjdk-17-jdk \
    openjdk-17-jre-headless \
    && rm -rf /var/lib/apt/lists/*

# Set JAVA_HOME
ENV JAVA_HOME=/usr/lib/jvm/java-17-openjdk-amd64
ENV PATH="${JAVA_HOME}/bin:${PATH}"

# Verify Java version
RUN java -version && javac -version

# ==================== Go 1.21 ====================
# Install Go 1.21 from official source
RUN wget -q https://go.dev/dl/go1.21.13.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.21.13.linux-amd64.tar.gz && \
    rm go1.21.13.linux-amd64.tar.gz

# Set Go environment
ENV GOPATH=/go
ENV GOROOT=/usr/local/go
ENV PATH="${GOROOT}/bin:${GOPATH}/bin:${PATH}"

# Verify Go version
RUN go version

# ==================== Rust ====================
# Install Rust using rustup
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain stable

# Add Rust to PATH
ENV PATH="/root/.cargo/bin:${PATH}"

# Verify Rust version
RUN rustc --version && cargo --version

# ==================== Node.js 18 ====================
# Install Node.js 18 from NodeSource
RUN curl -fsSL https://deb.nodesource.com/setup_18.x | bash - && \
    apt-get install -y --no-install-recommends nodejs && \
    rm -rf /var/lib/apt/lists/*

# Verify Node.js version
RUN node --version && npm --version

# ==================== Additional Tools ====================
# Install time measurement and process control tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    time \
    procps \
    cgroup-tools \
    && rm -rf /var/lib/apt/lists/*

# Create work directory
WORKDIR /app

# Copy the built daemon
COPY --from=builder /judgedaemon /judgedaemon

# Create directories for sandbox work
RUN mkdir -p /var/lib/judgehost/work /var/lib/judgehost/chroot

# Copy compile scripts
COPY judge/scripts/compile/*.sh /usr/local/bin/compile/
RUN chmod +x /usr/local/bin/compile/*.sh

# Create symlinks for easier access
RUN ln -sf /usr/local/bin/compile/cpp.sh /usr/local/bin/compile-cpp && \
    ln -sf /usr/local/bin/compile/c.sh /usr/local/bin/compile-c && \
    ln -sf /usr/local/bin/compile/python3.sh /usr/local/bin/compile-python3 && \
    ln -sf /usr/local/bin/compile/java.sh /usr/local/bin/compile-java && \
    ln -sf /usr/local/bin/compile/go.sh /usr/local/bin/compile-go && \
    ln -sf /usr/local/bin/compile/rust.sh /usr/local/bin/compile-rust && \
    ln -sf /usr/local/bin/compile/nodejs.sh /usr/local/bin/compile-nodejs && \
    ln -sf /usr/local/bin/compile/compile-runner.sh /usr/local/bin/compile-runner

# Set timezone
ENV TZ=UTC

# Health check script
RUN echo '#!/bin/bash\n\
echo "=== Compiler Versions ==="\n\
echo "g++: $(g++ --version | head -1)"\n\
echo "Python3: $(python3 --version)"\n\
echo "Java: $(java -version 2>&1 | head -1)"\n\
echo "Go: $(go version)"\n\
echo "Rust: $(rustc --version)"\n\
echo "Node.js: $(node --version)"\n\
' > /usr/local/bin/check-compilers && chmod +x /usr/local/bin/check-compilers

# Run health check on startup to verify all compilers
RUN /usr/local/bin/check-compilers

ENTRYPOINT ["/judgedaemon"]