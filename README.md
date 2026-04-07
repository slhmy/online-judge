# Online Judge Platform

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
│                    Backend Microservices (Go + gRPC)             │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐                │
│  │   Problem   │ │ Submission  │ │   Contest   │                │
│  │   Service   │ │   Service   │ │   Service   │                │
│  └─────────────┘ └─────────────┘ └─────────────┘                │
│  ┌─────────────┐ ┌─────────────┐                                 │
│  │Notification │ │    User     │                                 │
│  │   Service   │ │   Service   │                                 │
│  └─────────────┘ └─────────────┘                                 │
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
├── backend/           # Go microservices with gRPC
│   ├── cmd/           # Service entrypoints
│   ├── internal/      # Service implementations
│   ├── proto/         # Protobuf definitions
│   ├── gen/           # Generated code
│   └── migrations/    # Database migrations
├── frontend/          # Next.js frontend
│   ├── src/app/       # App Router pages
│   ├── src/components/
│   └── src/lib/
├── bff/               # Go Backend for Frontend
│   ├── cmd/
│   └── internal/
├── judge/             # Judging system
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
make run-problem    # Problem service on port 8002
make run-submission # Submission service on port 8003
make run-contest    # Contest service on port 8004
make run-notification # Notification service on port 8005
make run-user       # User service (default port 8002)
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
| BFF | 8080 | Go BFF layer |
| Identra (gRPC) | 50051 | Authentication service |
| Problem Service | 8002 | gRPC - Problem management |
| Submission Service | 8003 | gRPC - Submissions |
| Contest Service | 8004 | gRPC - Contests |
| Notification Service | 8005 | gRPC - WebSocket, notifications |
| User Service | configurable | gRPC - User management |
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
make test-problem-service
make test-submission-service
make test-contest-service
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