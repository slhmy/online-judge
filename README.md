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
│  ┌─────────────┐                                                 │
│  │Notification │                                                 │
│  │   Service   │                                                 │
│  └─────────────┘                                                 │
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
| BFF | Go, Gin/Fiber |
| Auth | Identra (JWT, OAuth, Email, Password) |
| Database | PostgreSQL 16 |
| Cache/Queue | Redis 7 |
| Object Storage | MinIO / S3 / Local |
| Message Queue | RabbitMQ |
| Judge | Docker, cgroups |

## Project Structure

```
.
├── backend/           # Go microservices with gRPC
│   ├── cmd/           # Service entrypoints
│   ├── internal/      # Service implementations
│   ├── proto/         # Protobuf definitions
│   └── gen/           # Generated code
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
└── docs/              # Architecture documentation
    └── architecture/
```

## Quick Start

### Prerequisites

- Go 1.21+
- Node.js 20+
- Docker & Docker Compose
- Buf (for protobuf)

### Development

```bash
# Start infrastructure services
make infra-up

# Generate protobuf code
make proto

# Run all services
make run

# Run tests
make test
```

## Services

| Service | Port | Description |
|---------|------|-------------|
| Frontend | 3000 | Next.js web application |
| BFF | 8080 | Go BFF layer |
| Identra | 8081 | Authentication service |
| Problem Service | 8002 | gRPC - Problem management |
| Submission Service | 8003 | gRPC - Submissions |
| Contest Service | 8004 | gRPC - Contests |
| Notification Service | 8005 | gRPC - WebSocket, notifications |
| Judge Daemon | - | Judge workers |

## Documentation

- [Backend Architecture](docs/architecture/backend.md)
- [Frontend Architecture](docs/architecture/frontend.md)
- [Judging System](docs/architecture/judging-system.md)

## License

MIT