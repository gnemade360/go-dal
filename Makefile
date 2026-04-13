.PHONY: help docker-up docker-down docker-reset podman-up podman-down podman-reset run test build clean

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

# Docker commands
docker-up: ## Start PostgreSQL container with Docker
	docker-compose up -d
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5
	@echo "PostgreSQL is ready!"

docker-down: ## Stop PostgreSQL container with Docker
	docker-compose down

docker-reset: ## Reset PostgreSQL container and data with Docker
	docker-compose down -v
	docker-compose up -d
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5
	@echo "PostgreSQL is ready with fresh data!"

# Podman commands
podman-up: ## Start PostgreSQL container with Podman
	./scripts/podman-postgres.sh start

podman-down: ## Stop PostgreSQL container with Podman
	./scripts/podman-postgres.sh stop

podman-reset: ## Reset PostgreSQL container and data with Podman
	./scripts/podman-postgres.sh reset
	./scripts/podman-postgres.sh start

podman-logs: ## Show PostgreSQL logs with Podman
	./scripts/podman-postgres.sh logs

podman-status: ## Show PostgreSQL status with Podman
	./scripts/podman-postgres.sh status

# Podman compose alternative (if podman-compose is installed)
podman-compose-up: ## Start PostgreSQL with podman-compose
	podman-compose -f podman-compose.yml up -d

podman-compose-down: ## Stop PostgreSQL with podman-compose
	podman-compose -f podman-compose.yml down

# Application commands
run: ## Run the Basic CRUD application
	DATABASE_URL="postgres://dal_user:dal_pass@localhost:5432/crud_db?sslmode=disable" \
	go run cmd/basic-crud/main.go

test: ## Run tests
	go test ./... -v

build: ## Build the Basic CRUD application
	go build -o bin/basic-crud cmd/basic-crud/main.go

clean: ## Clean build artifacts
	rm -rf bin/

deps: ## Install dependencies
	go mod download
	go mod tidy

# Quick start with Podman (default)
up: podman-up ## Start PostgreSQL (alias for podman-up)

down: podman-down ## Stop PostgreSQL (alias for podman-down)

reset: podman-reset ## Reset PostgreSQL (alias for podman-reset)

logs: podman-logs ## Show logs (alias for podman-logs)

status: podman-status ## Show status (alias for podman-status)