.PHONY: all proto build run test clean infra-up infra-down seed seed-docker seed-reset build-seed judge-runtime-image

# Default target
all: proto build

# ============================================
# Judge Runtime Image (Multi-language compilers)
# ============================================

judge-runtime-image:
	@echo "Building judge runtime Docker image with all compilers..."
	cd judge/scripts && chmod +x build-runtime-image.sh && ./build-runtime-image.sh
	@echo "Judge runtime image built successfully!"
	@echo ""
	@echo "Supported languages:"
	@echo "  - C++17 (g++)"
	@echo "  - C11 (gcc)"
	@echo "  - Python3"
	@echo "  - Java17"
	@echo "  - Go 1.21"
	@echo "  - Rust (stable)"
	@echo "  - Node.js 18"

judge-runtime-check:
	@echo "Verifying judge runtime compilers..."
	docker run --rm judge-runtime:latest /usr/local/bin/check-compilers

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
	@for f in backend/migrations/*.up.sql; do \
		echo "Applying $$f..."; \
		cat "$$f" | docker-compose exec -T postgres psql -U oj -d oj; \
	done

migrate-down:
	@echo "Running down migrations..."
	@for f in backend/migrations/*.down.sql; do \
		echo "Applying $$f..."; \
		cat "$$f" | docker-compose exec -T postgres psql -U oj -d oj; \
	done

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
	@echo "Running migrations and seeding..."
	@for f in backend/migrations/*.up.sql; do \
		cat "$$f" | docker-compose exec -T postgres psql -U oj -d oj 2>/dev/null || true; \
	done
	cd backend && go run ./cmd/seed

seed-reset:
	@echo "Resetting and re-seeding database..."
	docker-compose exec -T postgres psql -U oj -d oj -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	$(MAKE) migrate-up
	$(MAKE) seed

# Build seed binary (for Docker deployment)
build-seed:
	@echo "Building seed binary..."
	cd backend && go build -o bin/seed ./cmd/seed

# ============================================
# Testing
# ============================================

# Run all unit tests
test-backend:
	@echo "Running backend tests..."
	cd backend && go test ./... -v -race

test-bff:
	@echo "Running BFF tests..."
	cd bff && go test ./... -v -race

test-judge:
	@echo "Running judge tests..."
	cd judge && go test ./... -v -race

test-frontend:
	cd frontend && npm test

test: test-backend test-bff test-judge

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	cd backend && go test ./... -coverprofile=coverage.out -covermode=atomic
	cd backend && go tool cover -html=coverage.out -o coverage.html
	cd bff && go test ./... -coverprofile=coverage.out -covermode=atomic
	cd bff && go tool cover -html=coverage.out -o coverage.html
	cd judge && go test ./... -coverprofile=coverage.out -covermode=atomic
	cd judge && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage reports generated: backend/coverage.html, bff/coverage.html, judge/coverage.html"

# Run tests with coverage summary
test-coverage-summary:
	@echo "=== Backend Coverage ==="
	cd backend && go test ./... -coverprofile=coverage.out -covermode=atomic 2>/dev/null && go tool cover -func=coverage.out | tail -1
	@echo "=== BFF Coverage ==="
	cd bff && go test ./... -coverprofile=coverage.out -covermode=atomic 2>/dev/null && go tool cover -func=coverage.out | tail -1
	@echo "=== Judge Coverage ==="
	cd judge && go test ./... -coverprofile=coverage.out -covermode=atomic 2>/dev/null && go tool cover -func=coverage.out | tail -1

# Run specific package tests
test-problem-service:
	cd backend && go test ./internal/problem/... -v -race

test-submission-service:
	cd backend && go test ./internal/submission/... -v -race

test-contest-service:
	cd backend && go test ./internal/contest/... -v -race

test-notification-service:
	cd backend && go test ./internal/notification/... -v -race

test-user-service:
	cd backend && go test ./internal/user/... -v -race

test-bff-handlers:
	cd bff && go test ./internal/handler/... -v -race

test-judge-queue:
	cd judge && go test ./internal/queue/... -v -race

test-judge-validator:
	cd judge && go test ./internal/validator/... -v -race

test-judge-worker:
	cd judge && go test ./internal/worker/... -v -race

# ============================================
# Integration Tests (using mocks, no external services needed)
# ============================================

# Run integration tests for backend services
test-integration-backend:
	@echo "Running backend integration tests..."
	cd backend && go test ./... -v -race -run "Integration"

# Run integration tests for BFF handlers
test-integration-bff:
	@echo "Running BFF integration tests..."
	cd bff && go test ./... -v -race -run "Integration"

# Run integration tests for judge system
test-integration-judge:
	@echo "Running judge integration tests..."
	cd judge && go test ./... -v -race -run "Integration"

# Run all integration tests
test-integration: test-integration-backend test-integration-bff test-integration-judge
	@echo "All integration tests completed!"

# ============================================
# Extended Test Commands
# ============================================

# Run tests in short mode (skip slow tests)
test-short:
	cd backend && go test ./... -v -short -race
	cd bff && go test ./... -v -short -race
	cd judge && go test ./... -v -short -race

# Run tests with verbose output and timeout
test-verbose:
	cd backend && go test ./... -v -race -timeout 5m
	cd bff && go test ./... -v -race -timeout 5m
	cd judge && go test ./... -v -race -timeout 5m

# Generate test mocks (if using mockery)
generate-mocks:
	@echo "Generating test mocks..."
	cd backend && go generate ./...
	cd bff && go generate ./...
	cd judge && go generate ./...

# Clean test artifacts
test-clean:
	rm -f backend/coverage.out backend/coverage.html
	rm -f bff/coverage.out bff/coverage.html
	rm -f judge/coverage.out judge/coverage.html

# Run specific service tests with a prompt
test-service:
	@echo "Available services: problem, submission, contest, notification, user"
	@read -p "Enter service name: " svc; \
	case $$svc in \
		problem) cd backend && go test ./internal/problem/... -v -race ;; \
		submission) cd backend && go test ./internal/submission/... -v -race ;; \
		contest) cd backend && go test ./internal/contest/... -v -race ;; \
		notification) cd backend && go test ./internal/notification/... -v -race ;; \
		user) cd backend && go test ./internal/user/... -v -race ;; \
		*) echo "Unknown service: $$svc" ;; \
	esac

# Benchmark tests
test-bench:
	@echo "Running benchmark tests..."
	cd backend && go test ./... -bench=. -benchmem
	cd bff && go test ./... -bench=. -benchmem
	cd judge && go test ./... -bench=. -benchmem

# Run tests and generate HTML coverage report
test-coverage-html: test-coverage
	@echo "Opening coverage reports..."
	@echo "Backend: backend/coverage.html"
	@echo "BFF: bff/coverage.html"
	@echo "Judge: judge/coverage.html"

# Run tests with race detection and output to file
test-report:
	@echo "Generating test report..."
	cd backend && go test ./... -v -race -json > backend-test-report.json 2>&1 || true
	cd bff && go test ./... -v -race -json > bff-test-report.json 2>&1 || true
	cd judge && go test ./... -v -race -json > judge-test-report.json 2>&1 || true
	@echo "Test reports generated: backend-test-report.json, bff-test-report.json, judge-test-report.json"

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