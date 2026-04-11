# BFF Dockerfile
FROM golang:1.24-alpine AS builder

# Install git for go mod
RUN apk add --no-cache git

WORKDIR /app

# Set GOPRIVATE to avoid trying to fetch private modules
ENV GOPRIVATE=github.com/slhmy/online-judge/*
ENV GONOSUMDB=github.com/slhmy/online-judge/*
ENV GOINSECURE=github.com/slhmy/online-judge/*

# Copy gen module (pre-generated protobuf code)
COPY gen/ ./gen/

# Copy bff module
COPY bff/go.mod bff/go.sum ./
COPY bff/cmd ./cmd/
COPY bff/internal ./internal/

# Update replace directive for Docker build
RUN sed -i 's|=> ../gen|=> /app/gen|g' go.mod

# Download modules
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /bff ./cmd/bff

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bff /bff

ENV TZ=UTC

EXPOSE 8080

ENTRYPOINT ["/bff"]