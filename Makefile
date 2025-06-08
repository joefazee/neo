# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
BINARY_NAME=neo-api
BINARY_MIGRATE=neo-migrate
BINARY_ORACLE=neo-oracle

# Build directories
BUILD_DIR=bin

# Docker
DOCKER_COMPOSE=docker compose
DOCKER_COMPOSE_FILE=docker-compose.yml

# Database
DB_URL=postgres://neo:secret@localhost:5432/neo_dev?sslmode=disable

# Default environment
GO_ENV ?= development

.PHONY: all build clean test deps up down logs migrate-up migrate-down migrate-create help

## help: Show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## all: Build all binaries
all: clean build

## build: Build the application binaries
build:
	@echo "Building binaries..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v ./cmd/api
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_MIGRATE) -v ./cmd/migrations
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_ORACLE) -v ./cmd/oracle

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)

## deps: Download and tidy dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## test: Run tests
test:
	@echo "Running tests..."
	@GOFLAGS="-count=1" $(GOTEST) -v -race -cover ./...

## test-short: Run short tests (skips long tests)
test-short:
	@echo "Running short tests..."
	@GOFLAGS="-count=1" $(GOTEST) -v --short -race -cover ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@GOFLAGS="-count=1" $(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## format: Format Go code and organize imports
format:
	@echo "Formatting code..."
	@gofmt -s -w .
	@goimports -w .

## lint: Run comprehensive linting with golangci-lint
lint:
	@echo "Running comprehensive linting..."
	@golangci-lint run --config .golangci.yml --timeout 5m

## lint-fix: Run golangci-lint with auto-fix
lint-fix:
	@echo "Running linting with auto-fix..."
	@golangci-lint run --config .golangci.yml --fix

## security: Run comprehensive security analysis
security: security-gosec security-nancy security-staticcheck
	@echo "All security checks completed!"

## security-gosec: Run gosec security scanner
security-gosec:
	@echo "Running gosec security scanner..."
	@gosec -fmt=json -out=gosec-report.json ./...
	@gosec -fmt=text ./...

## security-nancy: Check for known vulnerabilities in dependencies
security-nancy:
	@echo "Checking for known vulnerabilities..."
	@go list -json -deps ./... | nancy sleuth

## security-staticcheck: Run staticcheck for additional security issues
security-staticcheck:
	@echo "Running staticcheck..."
	@staticcheck ./...

## up: Start all services with Docker Compose
up:
	@echo "Starting services..."
	$(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) up -d

## down: Stop all services
down:
	@echo "Stopping services..."
	$(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) down

## logs: Show logs from all services
logs:
	$(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) logs -f

## logs-api: Show API logs
logs-api:
	$(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) logs -f api

## ps: Show running containers
ps:
	$(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) ps

## migrate-up: Run database migrations up
migrate-up:
	@echo "Running migrations up..."
	@migrate -path migrations -database "$(DB_URL)" up

## migrate-down: Run database migrations down
migrate-down:
	@echo "Running migrations down..."
	migrate -path migrations -database "$(DB_URL)" down

## migrate-force: Force migration version (use with VERSION=N)
migrate-force:
	@echo "Forcing migration to version $(VERSION)..."
	migrate -path migrations -database "$(DB_URL)" force $(VERSION)

## migrate-version: Show current migration version
migrate-version:
	migrate -path migrations -database "$(DB_URL)" version

## migrate-create: Create new migration file (use with NAME=migration_name)
migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	migrate create -ext sql -dir migrations $(NAME)

## dev: Start development environment
dev: up migrate-up
	@echo "Development environment ready!"
	@echo "API: http://localhost:8080"
	@echo "Database: localhost:5432"
	@echo "Redis: localhost:6379"

## dev-down: Stop development environment
dev-down: down
	@echo "Development environment stopped"

## docker-build: Build Docker images
docker-build:
	@echo "Building Docker images..."
	$(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) build

## docker-rebuild: Rebuild Docker images without cache
docker-rebuild:
	@echo "Rebuilding Docker images..."
	$(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) build --no-cache

## reset-db: Reset database (WARNING: This will delete all data)
reset-db:
	@echo "WARNING: This will delete all database data!"
	@read -p "Are you sure? [y/N] " confirmation; \
	if [ "$$confirmation" = "y" ] || [ "$$confirmation" = "Y" ]; then \
		$(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) down -v; \
		$(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) up -d postgres redis; \
		sleep 5; \
		make migrate-up; \
		echo "Database reset complete!"; \
	else \
		echo "Operation cancelled."; \
	fi

## seed-db: Seed database with sample data
seed-db:
	@echo "Seeding database..."
	$(GOBUILD) -o $(BUILD_DIR)/seed ./cmd/seed
	./$(BUILD_DIR)/seed

## run-local: Run API locally (without Docker)
run-local: build
	@echo "Running API locally..."
	GO_ENV=$(GO_ENV) ./$(BUILD_DIR)/$(BINARY_NAME)

## install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "Installing Nancy vulnerability scanner..."
	@if [ "$$(uname)" = "Darwin" ]; then \
		brew install sonatype-nexus-community/nancy-tap/nancy || \
		(curl -L -o nancy $$(curl -s https://api.github.com/repos/sonatype-nexus-community/nancy/releases/latest | grep "browser_download_url.*darwin.amd64" | cut -d '"' -f 4) && \
		chmod +x nancy && \
		mv nancy $${GOPATH:-$$HOME/go}/bin/nancy); \
	else \
		curl -L -o nancy $$(curl -s https://api.github.com/repos/sonatype-nexus-community/nancy/releases/latest | grep "browser_download_url.*linux.amd64" | cut -d '"' -f 4) && \
		chmod +x nancy && \
		mv nancy $${GOPATH:-$$HOME/go}/bin/nancy; \
	fi
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@go install github.com/air-verse/air@latest
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@go install github.com/fzipp/gocyclo/cmd/gocyclo@latest

## quality: Run all quality checks
quality: format lint security test-coverage
	@echo "All quality checks passed!"

## complexity: Check cyclomatic complexity
complexity:
	@echo "Checking cyclomatic complexity..."
	@gocyclo -over 15 .

## deadcode: Find dead/unused code
deadcode:
	@echo "Finding dead code..."
	@golangci-lint run --disable-all --enable=deadcode,unused,varcheck,structcheck

## deps-check: Check for dependency issues
deps-check:
	@echo "Checking dependencies..."
	@go mod verify
	@go mod tidy -diff

## setup-quality: Setup quality and security tools for development
setup-quality: install-tools
	@echo "Setting up pre-commit hooks..."
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit install; \
		echo "Pre-commit hooks installed successfully!"; \
	else \
		echo "pre-commit not found. Install with: pip install pre-commit"; \
	fi
	@echo "Quality tools setup complete!"

## validate: Quick validation before commit
validate: format lint test
	@echo "Validation complete - ready to commit!"

## security-report: Generate comprehensive security report
security-report:
	@echo "Generating comprehensive security report..."
	@mkdir -p reports
	@gosec -fmt=json -out=reports/gosec-report.json ./...
	@gosec -fmt=html -out=reports/gosec-report.html ./...
	@go list -json -deps ./... | nancy sleuth -output=json > reports/nancy-report.json || true
	@echo "Security reports generated in ./reports/"

## deps-graph: Generate dependency graph
deps-graph:
	@echo "Generating dependency graph..."
	@go mod graph | dot -T svg -o reports/deps-graph.svg
	@echo "Dependency graph saved to reports/deps-graph.svg"

## vulnerability-check: Check for known vulnerabilities
vulnerability-check:
	@echo "Checking for vulnerabilities in dependencies..."
	@go list -json -deps ./... | nancy sleuth || true
	@govulncheck ./... || echo "govulncheck not available, install with: go install golang.org/x/vuln/cmd/govulncheck@latest"

## monitoring-up: Start monitoring stack (Prometheus + Grafana)
monitoring-up:
	$(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) --profile monitoring up -d

## monitoring-down: Stop monitoring stack
monitoring-down:
	$(DOCKER_COMPOSE) -f $(DOCKER_COMPOSE_FILE) --profile monitoring down

## docs-gen: Generate swagger documentation
docs-gen:
	@echo "Generating swagger documentation..."
	@swag init -g cmd/api/main.go -d ./ -o docs --parseDependency --parseInternal

## docs-clean: Clean generated documentation
docs-clean:
	@echo "Cleaning generated documentation..."
	@rm -rf docs/

## docs-validate: Validate swagger documentation
docs-validate: docs-gen
	@echo "Validating swagger documentation..."
	@swag fmt -g cmd/api/main.go
