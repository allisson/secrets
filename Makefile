.PHONY: help build run test lint clean migrate-up migrate-down docker-build docker-build-multiarch docker-inspect docker-scan docker-run-server docker-run-migrate mocks docs-lint docs-check-examples docs-check-metadata docs-check-release-tags

APP_NAME := app
BINARY_DIR := bin
BINARY := $(BINARY_DIR)/$(APP_NAME)
DOCKER_REGISTRY ?= allisson
DOCKER_IMAGE := $(DOCKER_REGISTRY)/secrets
DOCKER_TAG := latest
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_SHA ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")

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
	@go test -v -race -p 1 -coverprofile=coverage.out ./...
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

mocks: ## Regenerate mock implementations
	@echo "Regenerating mocks..."
	@mockery
	@echo "Mocks regenerated"

docs-check-examples: ## Validate JSON shapes used by docs examples
	@echo "Running docs example shape checks..."
	@python3 docs/tools/check_example_shapes.py

docs-check-metadata: ## Validate docs metadata and API markers
	@echo "Running docs metadata checks..."
	@python3 docs/tools/check_docs_metadata.py

docs-check-release-tags: ## Validate pinned release image tags in current docs
	@echo "Running docs release image tag checks..."
	@python3 docs/tools/check_release_image_tags.py

docs-lint: ## Run markdown lint and offline link checks (with auto-fix)
	@echo "Running markdownlint-cli2 (with auto-fix)..."
	@docker run --rm -v "$(PWD):/workdir" -w /workdir davidanson/markdownlint-cli2:v0.18.1 --fix README.md "docs/**/*.md" ".github/pull_request_template.md"
	@$(MAKE) docs-check-examples
	@$(MAKE) docs-check-metadata
	@$(MAKE) docs-check-release-tags
	@echo "Running lychee offline link checks..."
	@docker run --rm -v "$(PWD):/input" lycheeverse/lychee:latest --offline --include-fragments --no-progress "/input/README.md" "/input/docs/**/*.md" "/input/.github/pull_request_template.md"

# Database migrations
migrate-up: ## Run database migrations up
	@echo "Running migrations up..."
	@$(BINARY) migrate

migrate-down: ## Run database migrations down
	@echo "Rollback migrations not implemented in binary. Use golang-migrate CLI directly."

# Docker
docker-build: ## Build Docker image with version injection
	@echo "Building Docker image..."
	@echo "  Version: $(VERSION)"
	@echo "  Build Date: $(BUILD_DATE)"
	@echo "  Commit SHA: $(COMMIT_SHA)"
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg COMMIT_SHA=$(COMMIT_SHA) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		.
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG) and $(DOCKER_IMAGE):$(VERSION)"

docker-build-multiarch: ## Build and push multi-platform Docker image
	@echo "Building multi-platform Docker image..."
	@echo "  Version: $(VERSION)"
	@echo "  Build Date: $(BUILD_DATE)"
	@echo "  Commit SHA: $(COMMIT_SHA)"
	@echo "  Platforms: linux/amd64, linux/arm64"
	@docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg COMMIT_SHA=$(COMMIT_SHA) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		--push \
		.
	@echo "Multi-platform images pushed: $(DOCKER_IMAGE):$(DOCKER_TAG) and $(DOCKER_IMAGE):$(VERSION)"
	@echo "Note: Requires 'docker buildx' and authenticated registry access"

docker-inspect: ## Inspect Docker image metadata and labels
	@echo "Inspecting Docker image: $(DOCKER_IMAGE):$(DOCKER_TAG)"
	@echo ""
	@echo "=== Version Information ==="
	@docker inspect $(DOCKER_IMAGE):$(DOCKER_TAG) --format='Version: {{index .Config.Labels "org.opencontainers.image.version"}}'
	@docker inspect $(DOCKER_IMAGE):$(DOCKER_TAG) --format='Build Date: {{index .Config.Labels "org.opencontainers.image.created"}}'
	@docker inspect $(DOCKER_IMAGE):$(DOCKER_TAG) --format='Commit SHA: {{index .Config.Labels "org.opencontainers.image.revision"}}'
	@echo ""
	@echo "=== Security Information ==="
	@docker inspect $(DOCKER_IMAGE):$(DOCKER_TAG) --format='User: {{.Config.User}}'
	@docker inspect $(DOCKER_IMAGE):$(DOCKER_TAG) --format='Base Image: {{index .Config.Labels "org.opencontainers.image.base.name"}}'
	@echo ""
	@echo "=== Full Labels (JSON) ==="
	@docker inspect $(DOCKER_IMAGE):$(DOCKER_TAG) --format='{{json .Config.Labels}}' | jq .

docker-scan: ## Scan Docker image for vulnerabilities
	@echo "Scanning Docker image for vulnerabilities: $(DOCKER_IMAGE):$(DOCKER_TAG)"
	@if command -v trivy >/dev/null 2>&1; then \
		trivy image --severity HIGH,CRITICAL $(DOCKER_IMAGE):$(DOCKER_TAG); \
	else \
		echo ""; \
		echo "⚠️  Trivy not installed. Install with:"; \
		echo "  macOS:   brew install trivy"; \
		echo "  Linux:   https://aquasecurity.github.io/trivy/latest/getting-started/installation/"; \
		echo ""; \
		echo "Alternative: Use Docker Scout (built-in):"; \
		echo "  docker scout cves $(DOCKER_IMAGE):$(DOCKER_TAG)"; \
		echo ""; \
	fi

docker-run-server: docker-build ## Build and run Docker container (server)
	@echo "Running Docker container (server)..."
	@docker run --rm -p 8080:8080 \
		-e DB_DRIVER=postgres \
		-e DB_CONNECTION_STRING="postgres://user:password@host.docker.internal:5432/mydb?sslmode=disable" \
		$(DOCKER_IMAGE):$(DOCKER_TAG) server

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
