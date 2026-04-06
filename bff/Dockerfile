# BFF Dockerfile
FROM golang:1.24-alpine AS builder

# Install git for go mod
RUN apk add --no-cache git

WORKDIR /app

# Set GOPRIVATE to avoid trying to fetch private modules
ENV GOPRIVATE=github.com/online-judge/*
ENV GONOSUMDB=github.com/online-judge/*
ENV GOINSECURE=github.com/online-judge/*

# Copy backend module
COPY backend/go.mod backend/go.sum ./backend/
COPY backend/gen ./backend/gen/
COPY backend/proto ./backend/proto/

# Copy bff module
COPY bff/go.mod bff/go.sum ./
COPY bff/cmd ./cmd/
COPY bff/internal ./internal/

# Update replace directive for Docker build - only replace the path, not module name
RUN sed -i 's|=> ../backend|=> /app/backend|g' go.mod

# Debug: show go.mod and backend structure
RUN echo "=== go.mod ===" && cat go.mod && echo "=== backend structure ===" && ls -la /app/backend/ && ls -la /app/backend/gen/go/

# Tidy and download modules
RUN go mod tidy && go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /bff ./cmd/bff

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bff /bff

ENV TZ=UTC

EXPOSE 8080

ENTRYPOINT ["/bff"]