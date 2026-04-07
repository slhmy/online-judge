# Backend Service Dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git, buf CLI, and protoc plugins for protobuf generation
RUN apk add --no-cache git curl && \
    curl -sSL "https://github.com/bufbuild/buf/releases/download/v1.47.2/buf_$(uname -s)_$(uname -m)" -o /usr/local/bin/buf && \
    chmod +x /usr/local/bin/buf && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest

COPY go.mod go.sum ./
RUN go mod download

# Copy proto files and buf config for generation
COPY proto/ proto/
COPY buf.gen.yaml buf.work.yaml buf.lock ./

# Generate protobuf code
RUN buf generate

# Copy the rest of the source code
COPY . .

ARG SERVICE
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/${SERVICE}

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /server /server

ENV TZ=UTC

ENTRYPOINT ["/server"]