# Backend Service Dockerfile
# Build context must be the repo root (context: . in docker-compose)
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

# Copy gen module (pre-generated protobuf code)
COPY gen/ ./gen/

# Copy backend module and fix replace directive for Docker context
COPY backend/go.mod backend/go.sum ./backend/
WORKDIR /app/backend
RUN sed -i 's|=> ../gen|=> /app/gen|g' go.mod
RUN go mod download

# Copy backend source
WORKDIR /app
COPY backend/ ./backend/

ARG SERVICE
WORKDIR /app/backend
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/${SERVICE}

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /server /server

ENV TZ=UTC

ENTRYPOINT ["/server"]