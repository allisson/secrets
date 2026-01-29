.PHONY: help build run test lint clean migrate-up migrate-down docker-build docker-run

APP_NAME := app
BINARY_DIR := bin
BINARY := $(BINARY_DIR)/$(APP_NAME)
DOCKER_IMAGE := go-project-template
DOCKER_TAG := latest

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BINARY_DIR)
	@go build -o $(BINARY) ./cmd/app
	@echo "Binary created at $(BINARY)"

run-server: build ## Build and run the HTTP server
	@echo "Running server..."
	@$(BINARY) server

run-worker: build ## Build and run the worker
	@echo "Running worker..."
	@$(BINARY) worker

run-migrate: build ## Build and run database migrations
	@echo "Running migrations..."
	@$(BINARY) migrate

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

test-with-db: test-db-up test test-db-down ## Run tests with test databases

test-db-up: ## Start test databases
	@echo "Starting test databases..."
	@docker compose -f docker-compose.test.yml up -d
	@echo "Waiting for databases to be ready..."
	@sleep 10

test-db-down: ## Stop test databases
	@echo "Stopping test databases..."
	@docker compose -f docker-compose.test.yml down -v

test-coverage: test ## Run tests and show coverage in browser
	@go tool cover -html=coverage.out

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run -v --fix

clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR)
	@rm -f coverage.out
	@echo "Clean complete"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Database migrations
migrate-up: ## Run database migrations up
	@echo "Running migrations up..."
	@$(BINARY) migrate

migrate-down: ## Run database migrations down
	@echo "Rollback migrations not implemented in binary. Use golang-migrate CLI directly."

# Docker
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

docker-run-server: docker-build ## Build and run Docker container (server)
	@echo "Running Docker container (server)..."
	@docker run --rm -p 8080:8080 \
		-e DB_DRIVER=postgres \
		-e DB_CONNECTION_STRING="postgres://user:password@host.docker.internal:5432/mydb?sslmode=disable" \
		$(DOCKER_IMAGE):$(DOCKER_TAG) server

docker-run-worker: docker-build ## Build and run Docker container (worker)
	@echo "Running Docker container (worker)..."
	@docker run --rm \
		-e DB_DRIVER=postgres \
		-e DB_CONNECTION_STRING="postgres://user:password@host.docker.internal:5432/mydb?sslmode=disable" \
		$(DOCKER_IMAGE):$(DOCKER_TAG) worker

docker-run-migrate: docker-build ## Build and run Docker container (migrate)
	@echo "Running Docker container (migrate)..."
	@docker run --rm \
		-e DB_DRIVER=postgres \
		-e DB_CONNECTION_STRING="postgres://user:password@host.docker.internal:5432/mydb?sslmode=disable" \
		$(DOCKER_IMAGE):$(DOCKER_TAG) migrate

# Development
dev-postgres: ## Start PostgreSQL in Docker for development
	@docker run --name dev-postgres -d \
		-e POSTGRES_USER=user \
		-e POSTGRES_PASSWORD=password \
		-e POSTGRES_DB=mydb \
		-p 5432:5432 \
		postgres:16-alpine

dev-mysql: ## Start MySQL in Docker for development
	@docker run --name dev-mysql -d \
		-e MYSQL_ROOT_PASSWORD=rootpassword \
		-e MYSQL_DATABASE=mydb \
		-e MYSQL_USER=user \
		-e MYSQL_PASSWORD=password \
		-p 3306:3306 \
		mysql:8.0

dev-stop: ## Stop development databases
	@docker stop dev-postgres dev-mysql || true
	@docker rm dev-postgres dev-mysql || true

.DEFAULT_GOAL := help
