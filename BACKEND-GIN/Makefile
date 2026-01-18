.PHONY: all build run test clean dev docker-up docker-down lint fmt help

# Variables
APP_NAME := chatbox-gin
MAIN_PATH := cmd/server/main.go
BUILD_DIR := bin
GO := go

# Default target
all: build

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	@$(GO) build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

# Run the application
run: build
	@./$(BUILD_DIR)/$(APP_NAME)

# Run in development mode with hot reload
dev:
	@echo "Starting in development mode..."
	@air -c .air.toml

# Run tests
test:
	@echo "Running tests..."
	@$(GO) test -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@$(GO) test -v -race -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# Format code
fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# Start Docker services
docker-up:
	@echo "Starting Docker services..."
	@docker-compose up -d

# Stop Docker services
docker-down:
	@echo "Stopping Docker services..."
	@docker-compose down

# View Docker logs
docker-logs:
	@docker-compose logs -f

# Seed database
seed:
	@echo "Seeding database..."
	@$(GO) run scripts/seed/main.go

# Database migration
migrate-up:
	@echo "Running migrations..."
	@psql -h localhost -U postgres -d chatbox-gin -f migrations/001_init.sql

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  run           - Build and run the application"
	@echo "  dev           - Run with hot reload (requires air)"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  clean         - Clean build artifacts"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  docker-up     - Start Docker services"
	@echo "  docker-down   - Stop Docker services"
	@echo "  docker-logs   - View Docker logs"
	@echo "  seed          - Seed database"
	@echo "  migrate-up    - Run database migrations"
	@echo "  help          - Show this help message"