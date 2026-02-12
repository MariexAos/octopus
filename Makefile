# Short Link Service Makefile

.PHONY: all build run test clean docker-build docker-run deps swagger help test-coverage test-race coverage-html coverage-check

# Variables
APP_NAME=octopus
CMD_PATH=./cmd/server
BUILD_DIR=./bin
DOCKER_IMAGE=$(APP_NAME):latest
DOCKER_COMPOSE_FILE=docker-compose.yaml
COVERAGE_THRESHOLD=80

# Build
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) $(CMD_PATH)/main.go

# Run
run:
	@echo "Running $(APP_NAME)..."
	@go run $(CMD_PATH)/main.go

# Test
test:
	@echo "Running tests..."
	@go test -v ./...

# Test with race detection
test-race:
	@echo "Running tests with race detection..."
	@go test -v -race ./...

# Test with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./internal/... ./pkg/...
	@go tool cover -func=coverage.out

# Generate HTML coverage report
coverage-html:
	@echo "Generating HTML coverage report..."
	@go test -coverprofile=coverage.out ./internal/... ./pkg/...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Check coverage threshold
coverage-check:
	@echo "Checking coverage threshold ($(COVERAGE_THRESHOLD)%)..."
	@go test -coverprofile=coverage.out ./internal/... ./pkg/... 2>/dev/null
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$COVERAGE < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "Coverage $$COVERAGE% is below $(COVERAGE_THRESHOLD)% threshold"; \
		exit 1; \
	else \
		echo "Coverage $$COVERAGE% meets $(COVERAGE_THRESHOLD)% threshold"; \
	fi

# Clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Swagger
swagger:
	@echo "Generating swagger docs..."
	@swag init -g $(CMD_PATH)/main.go -o ./api/swagger

# Docker build
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE) -f deployments/docker/Dockerfile .

# Docker run
docker-run:
	@echo "Running Docker container..."
	@docker run -p 8080:8080 --env-file deployments/docker/.env $(DOCKER_IMAGE)

# Docker compose
docker-compose-up:
	@echo "Starting services with Docker Compose..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d

docker-compose-down:
	@echo "Stopping Docker Compose services..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# Migrate database
migrate:
	@echo "Running database migration..."
	@mysql -h localhost -u root -p < scripts/migration.sql

# Development mode (auto-reload)
dev:
	@echo "Starting development server with air..."
	@air

help:
	@echo "Available targets:"
	@echo "  make build         - Build the application"
	@echo "  make run           - Run the application"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make deps          - Download and tidy dependencies"
	@echo "  make swagger       - Generate swagger docs"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-run    - Run Docker container"
	@echo "  make fmt           - Format code"
	@echo "  make lint          - Run linter"
	@echo "  make migrate       - Run database migration"
	@echo "  make dev           - Start dev server with air"
	@echo "  make help          - Show this help message"

.DEFAULT_GOAL := help
