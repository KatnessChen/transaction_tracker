.PHONY: build run test clean docker-build docker-run help

# Variables
BINARY_NAME=price-service
DOCKER_IMAGE=price-service:latest

# Default target
all: build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) main.go

# Run the application with local environment
run: build
	@echo "Starting $(BINARY_NAME) with local environment..."
	cp .env.local .env
	./$(BINARY_NAME)

# Run with development environment
run-dev:
	@echo "Starting $(BINARY_NAME) with development environment..."
	@if [ -f .env.local ]; then \
		export $$(cat .env.local | xargs) && go run main.go; \
	else \
		echo "No .env.local found, using default environment variables"; \
		go run main.go; \
	fi

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -cover ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -f .env

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

# Run with Docker Compose
docker-run:
	@echo "Starting services with Docker Compose..."
	docker-compose up --build

# Stop Docker Compose
docker-stop:
	@echo "Stopping Docker Compose services..."
	docker-compose down

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Start Redis for local development
redis-start:
	@echo "Starting Redis in Docker..."
	docker run -d --name price-service-redis -p 6379:6379 redis:7-alpine

# Stop Redis
redis-stop:
	@echo "Stopping Redis..."
	docker stop price-service-redis || true
	docker rm price-service-redis || true

# API examples
example-health:
	@echo "Testing health endpoint..."
	curl -s http://localhost:8081/health | jq .

example-current-prices:
	@echo "Testing current prices endpoint..."
	curl -s -H "X-API-Key: dev-api-key-12345" \
		"http://localhost:8081/api/v1/price/current/symbols?symbols=AAPL,GOOGL,MSFT" | jq .

example-historical:
	@echo "Testing historical prices endpoint..."
	curl -s -H "X-API-Key: dev-api-key-12345" \
		"http://localhost:8081/api/v1/price/historical/symbol?symbol=AAPL&from=2025-01-01&to=2025-07-17&resolution=daily" | jq .

# Help
help:
	@echo "Available targets:"
	@echo "  build           - Build the application"
	@echo "  run             - Run with local environment (.env.local)"
	@echo "  run-dev         - Run with development environment (.env.local)"
	@echo "  test            - Run tests"
	@echo "  test-coverage   - Run tests with coverage"
	@echo "  fmt             - Format code"
	@echo "  lint            - Run linter"
	@echo "  clean           - Clean build artifacts"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-run      - Start with Docker Compose"
	@echo "  docker-stop     - Stop Docker Compose"
	@echo "  deps            - Install dependencies"
	@echo "  redis-start     - Start Redis in Docker"
	@echo "  redis-stop      - Stop Redis"
	@echo "  example-*       - Test API endpoints"
	@echo "  help            - Show this help"
