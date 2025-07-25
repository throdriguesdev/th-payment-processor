# TH Payment Processor Makefile

.PHONY: help build run test clean init deploy logs

# Default target
help:
	@echo "Available targets:"
	@echo "  help     - Show this help message"
	@echo "  build    - Build the application binary"
	@echo "  run      - Run the application locally"
	@echo "  test     - Run all tests"
	@echo "  init     - Initialize complete environment (processors + backend)"
	@echo "  deploy   - Deploy only the backend services"
	@echo "  clean    - Clean up all services and resources"
	@echo "  logs     - Show backend logs"
	@echo "  stress   - Run stress tests"

# Build the application
build:
	@echo "Building th_payment_processor..."
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/th_payment_processor ./cmd/server

# Run the application locally (for development)
run:
	@echo "Running th_payment_processor locally..."
	@go run ./cmd/server

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...
	@echo "Running integration tests..."
	@cd scripts && ./test_payments.sh
	@cd scripts && ./test_processors.sh

# Initialize complete environment
init:
	@echo "Initializing complete environment..."
	@cd scripts && ./init.sh

# Deploy backend services only
deploy:
	@echo "Deploying backend services..."
	@cd deployments && docker compose up -d --build

# Clean up all services
clean:
	@echo "Cleaning up environment..."
	@cd scripts && ./cleanup.sh

# Show backend logs
logs:
	@echo "Showing backend logs..."
	@cd deployments && docker compose logs -f

# Run stress tests
stress:
	@echo "Running stress tests..."
	@cd scripts && ./stress_test.sh 10 50

# Development convenience targets
dev-deps:
	@echo "Installing development dependencies..."
	@go mod download
	@go mod tidy

format:
	@echo "Formatting code..."
	@go fmt ./...

lint:
	@echo "Running linter..."
	@golangci-lint run

# Docker convenience targets
docker-build:
	@echo "Building Docker image..."
	@cd deployments && docker compose build

docker-clean:
	@echo "Cleaning Docker resources..."
	@docker system prune -f

# Monitoring targets
status:
	@echo "Checking service status..."
	@cd deployments && docker compose ps

health:
	@echo "Checking health endpoints..."
	@curl -s http://localhost:9999/payments-summary | jq || echo "Backend not available"
	@curl -s http://localhost:8001/payments/service-health | jq || echo "Default processor not available"
	@curl -s http://localhost:8002/payments/service-health | jq || echo "Fallback processor not available"