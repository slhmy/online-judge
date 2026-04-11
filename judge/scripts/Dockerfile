# Judge Runtime Image - Contains all compilers for submission compilation/execution
# This image is used by the judge daemon to run submissions in isolated containers

# Build runguard (precise resource measurement wrapper)
# runguard uses only stdlib, so we strip the gen replace directive
FROM golang:1.24-alpine AS runguard-builder
WORKDIR /build
COPY go.mod go.sum ./
RUN sed -i '/replace.*\/gen/d' go.mod && go mod download
COPY cmd/runguard/ ./cmd/runguard/
RUN CGO_ENABLED=0 GOOS=linux go build -o /runguard ./cmd/runguard

# Runtime stage with all compilers
FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

# Install base dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    curl \
    wget \
    gnupg \
    software-properties-common \
    && rm -rf /var/lib/apt/lists/*

# ==================== C++17 (g++) ====================
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    g++ \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# ==================== Python3 ====================
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    python3-pip \
    python3-dev \
    && rm -rf /var/lib/apt/lists/*

RUN pip3 install --no-cache-dir \
    numpy \
    scipy \
    sympy \
    pypy3 || true

# ==================== Java 17 ====================
RUN apt-get update && apt-get install -y --no-install-recommends \
    openjdk-17-jdk \
    openjdk-17-jre-headless \
    && rm -rf /var/lib/apt/lists/*

ENV JAVA_HOME=/usr/lib/jvm/java-17-openjdk-amd64
ENV PATH="${JAVA_HOME}/bin:${PATH}"

# ==================== Go 1.21 ====================
RUN wget -q https://go.dev/dl/go1.21.13.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.21.13.linux-amd64.tar.gz && \
    rm go1.21.13.linux-amd64.tar.gz

ENV GOPATH=/go
ENV GOROOT=/usr/local/go
ENV PATH="${GOROOT}/bin:${GOPATH}/bin:${PATH}"

# ==================== Rust ====================
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain stable
ENV PATH="/root/.cargo/bin:${PATH}"

# ==================== Node.js 18 ====================
RUN curl -fsSL https://deb.nodesource.com/setup_18.x | bash - && \
    apt-get install -y --no-install-recommends nodejs && \
    rm -rf /var/lib/apt/lists/*

# ==================== Additional Tools ====================
RUN apt-get update && apt-get install -y --no-install-recommends \
    time \
    procps \
    cgroup-tools \
    && rm -rf /var/lib/apt/lists/*

# Copy compile scripts
COPY scripts/compile/*.sh /usr/local/bin/compile/
RUN chmod +x /usr/local/bin/compile/*.sh

RUN ln -sf /usr/local/bin/compile/cpp.sh /usr/local/bin/compile-cpp && \
    ln -sf /usr/local/bin/compile/c.sh /usr/local/bin/compile-c && \
    ln -sf /usr/local/bin/compile/python3.sh /usr/local/bin/compile-python3 && \
    ln -sf /usr/local/bin/compile/java.sh /usr/local/bin/compile-java && \
    ln -sf /usr/local/bin/compile/go.sh /usr/local/bin/compile-go && \
    ln -sf /usr/local/bin/compile/rust.sh /usr/local/bin/compile-rust && \
    ln -sf /usr/local/bin/compile/nodejs.sh /usr/local/bin/compile-nodejs && \
    ln -sf /usr/local/bin/compile/compile-runner.sh /usr/local/bin/compile-runner

# Copy runguard binary for precise resource measurement
COPY --from=runguard-builder /runguard /usr/local/bin/runguard

WORKDIR /workspace

ENV TZ=UTC
