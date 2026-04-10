# Online Judge Platform

[![CI](https://github.com/slhmy/online-judge/actions/workflows/ci.yml/badge.svg)](https://github.com/slhmy/online-judge/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/online-judge/backend)](https://goreportcard.com/report/github.com/online-judge/backend)

A modernized online judge platform for competitive programming, built with microservices architecture.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Frontend (Next.js 14)                         │
│                    + Go BFF Layer                                │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                    Identra Auth Service                          │
│                    (OAuth, Email, Password, JWT)                 │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│              Backend - Unified gRPC Server (Go)                  │
│  All domain services run in a single process on port 8002:       │
│  Problem · Submission · Contest · User · Notification · Judge    │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                    Judge System (DOMjudge-style)                 │
│  - Docker + cgroups sandboxing                                   │
│  - Multi-language support                                        │
│  - Lazy judging with verdict priority                            │
└─────────────────────────────────────────────────────────────────┘
```

## Technology Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.21+, Buf, gRPC-Gateway |
| Frontend | Next.js 14, TypeScript, Tailwind CSS |
| BFF | Go, Gin |
| Auth | Identra (JWT, OAuth, Email, Password) |
| Database | PostgreSQL 16 |
| Cache/Queue | Redis 7 |
| Object Storage | MinIO / S3 / Local |
| Judge | Docker, cgroups |

## Project Structure

```
.
├── proto/             # Protobuf definitions (all services)
├── gen/               # Generated Go code (buf generate proto)
├── backend/           # Unified Go gRPC server
│   ├── cmd/server/    # Server entrypoint (all services in one process)
│   ├── cmd/seed/      # Database seed tool
│   ├── internal/      # Service implementations
│   └── migrations/    # Database migrations
├── frontend/          # Next.js frontend
│   ├── src/app/       # App Router pages
│   ├── src/components/
│   └── src/lib/
├── bff/               # Go Backend for Frontend (chi HTTP → gRPC)
│   ├── cmd/
│   └── internal/
├── judge/             # Judging daemon
│   ├── cmd/
│   └── internal/
├── configs/           # Configuration files
└── docs/              # Architecture documentation
    └── architecture/
```

## Quick Start

### Prerequisites

- Go 1.21+
- Node.js 20+
- Docker & Docker Compose
- Buf (for protobuf)
- golang-migrate (for database migrations)

### Development

```bash
# Start infrastructure only (Postgres, Redis, MinIO, Identra)
make infra-up

# Run database migrations
make migrate-up

# Seed database with sample data
make seed

# Generate protobuf code
make proto

# Run all services with Docker (full stack)
make run

# Or run services individually for development:
make run-bff        # BFF on port 8080
make run-backend    # Unified backend gRPC server on port 8002
make run-judge      # Judge daemon
make run-frontend   # Frontend on port 3000

# Run tests
make test

# View logs
make full-logs
```

### Stopping Services

```bash
# Stop full stack
make full-down

# Stop infrastructure only
make infra-down
```

## Services

| Service | Port | Description |
|---------|------|-------------|
| Frontend | 3000 | Next.js web application |
| BFF | 8080 | Go BFF layer (chi HTTP router) |
| Identra (gRPC) | 50051 | Authentication service |
| Backend (gRPC) | 8002 | Unified server: Problem, Submission, Contest, User, Notification, Judge |
| Judge Daemon | - | Judge workers |
| PostgreSQL | 5432 | Database |
| Redis | 6379 | Cache & message queue |
| MinIO | 9000/9001 | Object storage (API/Console) |

## Judge Runtime

The judge system supports multiple programming languages:

```bash
# Build the judge runtime Docker image with all compilers
make judge-runtime-image

# Verify installed compilers
make judge-runtime-check
```

Supported languages:
- C++17 (g++)
- C11 (gcc)
- Python3
- Java17
- Go 1.21
- Rust (stable)
- Node.js 18

## Testing

```bash
# Run all unit tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
make test-integration

# Run tests for specific service
make test-problem
make test-submission
make test-contest
```

## Database Management

```bash
# Run migrations
make migrate-up

# Rollback migrations
make migrate-down

# Create new migration
make migrate-create

# Reset and re-seed database
make seed-reset
```

## Environment Configuration

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

Key environment variables:
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection
- `IDENTRA_GRPC_HOST` - Identra gRPC service address
- `PROBLEM_SERVICE_ADDR`, `SUBMISSION_SERVICE_ADDR`, etc. - Backend service addresses
- `S3_*` - MinIO/S3 configuration for object storage

## Documentation

- [Backend Architecture](docs/architecture/backend.md)
- [Frontend Architecture](docs/architecture/frontend.md)
- [Judging System](docs/architecture/judging-system.md)

## License

MIT