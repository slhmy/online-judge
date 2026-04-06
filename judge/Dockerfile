# Judge Daemon Dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /judgedaemon ./cmd/judgedaemon

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata docker-cli

WORKDIR /app

COPY --from=builder /judgedaemon /judgedaemon

ENV TZ=UTC

ENTRYPOINT ["/judgedaemon"]