.PHONY: all proto build run test clean infra-up infra-down seed seed-docker seed-reset build-seed

# Default target
all: proto build

# ============================================
# Run Services
# ============================================

run:
	@echo "Starting all services..."
	docker-compose -f docker-compose.full.yaml up -d
	@echo "All services started. Use 'make full-logs' to view logs."

# ============================================
# Protobuf Generation
# ============================================

proto:
	@echo "Generating protobuf code..."
	cd backend && buf generate

proto-lint:
	cd backend && buf lint

proto-format:
	cd backend && buf format -w

# ============================================
# Build
# ============================================

build-backend:
	@echo "Building backend services..."
	cd backend && go build -o bin/problem-service ./cmd/problem-service
	cd backend && go build -o bin/submission-service ./cmd/submission-service
	cd backend && go build -o bin/contest-service ./cmd/contest-service
	cd backend && go build -o bin/notification-service ./cmd/notification-service
	cd backend && go build -o bin/user-service ./cmd/user-service

build-bff:
	@echo "Building BFF..."
	cd bff && go build -o bin/bff ./cmd/bff

build-judge:
	@echo "Building judge system..."
	cd judge && go build -o bin/judgedaemon ./cmd/judgedaemon

build-frontend:
	@echo "Building frontend..."
	cd frontend && npm run build

build: build-backend build-bff build-judge

# ============================================
# Development
# ============================================

run-frontend:
	cd frontend && npm run dev

run-bff:
	cd bff && go run ./cmd/bff

run-problem:
	cd backend && go run ./cmd/problem-service

run-submission:
	cd backend && go run ./cmd/submission-service

run-contest:
	cd backend && go run ./cmd/contest-service

run-notification:
	cd backend && go run ./cmd/notification-service

run-user:
	cd backend && go run ./cmd/user-service

run-judge:
	cd judge && go run ./cmd/judgedaemon

# ============================================
# Infrastructure
# ============================================

infra-up:
	docker-compose up -d

infra-down:
	docker-compose down

infra-logs:
	docker-compose logs -f

infra-ps:
	docker-compose ps

# Full stack with all services
full-up:
	docker-compose -f docker-compose.full.yaml up -d

full-down:
	docker-compose -f docker-compose.full.yaml down

full-logs:
	docker-compose -f docker-compose.full.yaml logs -f

# ============================================
# Database
# ============================================

migrate-up:
	@echo "Running migrations..."
	migrate -path backend/migrations -database "postgres://oj:oj@localhost:5432/oj?sslmode=disable" up

migrate-down:
	migrate -path backend/migrations -database "postgres://oj:oj@localhost:5432/oj?sslmode=disable" down

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir backend/migrations -seq $$name

# ============================================
# Seeding
# ============================================

seed:
	@echo "Running database seed..."
	cd backend && go run ./cmd/seed

seed-docker:
	@echo "Running database seed in Docker..."
	docker-compose exec problem-service sh -c "cd /app && ./seed" || \
		cd backend && go run ./cmd/seed

seed-reset:
	@echo "Resetting and re-seeding database..."
	migrate -path backend/migrations -database "postgres://oj:oj@localhost:5432/oj?sslmode=disable" down
	migrate -path backend/migrations -database "postgres://oj:oj@localhost:5432/oj?sslmode=disable" up
	$(MAKE) seed

# Build seed binary (for Docker deployment)
build-seed:
	@echo "Building seed binary..."
	cd backend && go build -o bin/seed ./cmd/seed

# ============================================
# Testing
# ============================================

test-backend:
	cd backend && go test ./... -v

test-bff:
	cd bff && go test ./... -v

test-judge:
	cd judge && go test ./... -v

test-frontend:
	cd frontend && npm test

test: test-backend test-bff test-judge

# ============================================
# Docker
# ============================================

docker-build:
	docker-compose -f docker-compose.full.yaml build

docker-run:
	docker-compose -f docker-compose.full.yaml up -d

docker-stop:
	docker-compose -f docker-compose.full.yaml down

docker-logs:
	docker-compose -f docker-compose.full.yaml logs -f

# ============================================
# Clean
# ============================================

clean:
	rm -rf backend/bin
	rm -rf bff/bin
	rm -rf judge/bin
	rm -rf backend/gen
	cd frontend && rm -rf .next node_modules
	find . -name "*.log" -delete

# ============================================
# Install Dependencies
# ============================================

install-backend:
	cd backend && go mod download

install-frontend:
	cd frontend && npm install

install: install-backend install-frontend