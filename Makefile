# ============================================================
# Nucleus Remote Access Portal — Makefile
# Usage: make <target>
# ============================================================

.PHONY: help up down dev build test migrate seed logs clean \
        api-dev web-dev agent-build windows-helper-build \
        api-test db-reset lint format ci

# Default: show help
.DEFAULT_GOAL := help

help: ## Show this help message
	@echo ""
	@echo "  Nucleus Remote Access Portal"
	@echo "  =============================="
	@awk 'BEGIN {FS = ":.*##"; printf "\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""

# ---- Environment Setup ----------------------------------------

setup: ## First-time setup: copy .env.example, install deps
	@if [ ! -f .env ]; then cp .env.example .env; echo "Created .env from .env.example — edit it now!"; fi
	@cd apps/web && npm install
	@echo "Setup complete. Run 'make up' to start all services."

# ---- Docker Compose -------------------------------------------

up: ## Start all services (Docker Compose)
	docker compose up -d
	@echo ""
	@echo "  Services started:"
	@echo "  → Web Portal:   http://localhost:3000"
	@echo "  → API:          http://localhost:8080"
	@echo "  → API Docs:     http://localhost:8080/api/v1/health"
	@echo "  → PostgreSQL:   localhost:5432"
	@echo "  → Redis:        localhost:6379"
	@echo ""

down: ## Stop all services
	docker compose down

down-volumes: ## Stop all services and remove volumes (WARNING: destroys data)
	docker compose down -v

restart: ## Restart all services
	docker compose restart

logs: ## Tail logs for all services
	docker compose logs -f

logs-api: ## Tail API logs
	docker compose logs -f api

logs-web: ## Tail web logs
	docker compose logs -f web

logs-agents: ## Tail agent logs
	docker compose logs -f agent-n1001 agent-n1002 agent-n1003 agent-n1004

ps: ## Show running services
	docker compose ps

# ---- Development (without Docker) ----------------------------

dev: ## Start all services in dev mode (requires local Go + Node)
	@echo "Starting dev services..."
	@make -j3 api-dev web-dev

api-dev: ## Start Go API with hot reload (requires air)
	@cd apps/api && air

web-dev: ## Start Next.js dev server
	@cd apps/web && npm run dev

# ---- Build ---------------------------------------------------

build: ## Build all Docker images
	docker compose build

build-api: ## Build API Docker image only
	docker compose build api

build-web: ## Build web Docker image only
	docker compose build web

build-agent: ## Build agente-rs Docker image (linux/amd64 + linux/arm/v7)
	docker buildx build \
		--platform linux/amd64,linux/arm/v7 \
		-t agente-rs:latest \
		apps/agent/

build-windows-helper: ## Build Windows helper CLI
	@cd tools/windows-helper && GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o nucleus-helper.exe ./cmd
	@echo "Built: tools/windows-helper/nucleus-helper.exe"

# ---- Database -----------------------------------------------

migrate: ## Run database migrations
	@echo "Running migrations..."
	@docker compose exec postgres psql -U $${POSTGRES_USER:-nucleus} -d $${POSTGRES_DB:-nucleus_portal} \
		-f /migrations/001_initial_schema.sql \
		-f /migrations/002_session_functions.sql
	@echo "Migrations complete."

seed: ## Run database seeds (test data)
	@echo "Running seeds..."
	@docker compose exec -e DATABASE_URL=postgresql://$${POSTGRES_USER:-nucleus}:$${POSTGRES_PASSWORD:-nucleus_dev}@localhost:5432/$${POSTGRES_DB:-nucleus_portal} postgres bash /seeds/run_seeds.sh
	@echo "Seeds complete."

db-reset: ## Drop and recreate database, run migrations + seeds
	@echo "WARNING: This will destroy all data. Press Ctrl+C to cancel."
	@sleep 3
	docker compose exec postgres psql -U $${POSTGRES_USER:-nucleus} -c "DROP DATABASE IF EXISTS $${POSTGRES_DB:-nucleus_portal};"
	docker compose exec postgres psql -U $${POSTGRES_USER:-nucleus} -c "CREATE DATABASE $${POSTGRES_DB:-nucleus_portal};"
	@make migrate
	@make seed

db-shell: ## Open PostgreSQL shell
	docker compose exec postgres psql -U $${POSTGRES_USER:-nucleus} -d $${POSTGRES_DB:-nucleus_portal}

redis-shell: ## Open Redis CLI
	docker compose exec redis redis-cli

# ---- Testing -------------------------------------------------

test: ## Run all tests
	@make api-test
	@make web-test

api-test: ## Run Go API tests
	@cd apps/api && go test ./... -v -cover

agent-test: ## Run Go agent tests
	@cd apps/agent && go test ./... -v -cover

web-test: ## Run frontend tests
	@cd apps/web && npm test -- --passWithNoTests

test-integration: ## Run integration tests (requires running services)
	@cd apps/api && go test ./... -v -tags=integration

# ---- Code Quality --------------------------------------------

lint: ## Run linters
	@cd apps/api && go vet ./...
	@cd apps/agent && go vet ./...
	@cd tools/windows-helper && go vet ./...
	@cd apps/web && npm run lint

format: ## Format all code
	@cd apps/api && gofmt -w .
	@cd apps/agent && gofmt -w .
	@cd tools/windows-helper && gofmt -w .
	@cd apps/web && npm run format

# ---- Simulation ---------------------------------------------

simulate-agents: ## Start 4 simulated Nucleus agents
	docker compose up -d agent-n1001 agent-n1002 agent-n1003 agent-n1004
	@echo "Agents N-1001, N-1002, N-1003, N-1004 started."

simulate-offline-n1003: ## Put N-1003 in offline state
	docker compose stop agent-n1003
	@echo "N-1003 is now offline."

simulate-online-n1003: ## Bring N-1003 back online
	docker compose start agent-n1003
	@echo "N-1003 is now online."

# ---- GitHub -------------------------------------------------

git-init: ## Initialize git repo and create first commit
	@if [ ! -d .git ]; then git init; fi
	git add .
	git commit -m "feat: initial scaffold for Nucleus Remote Access Portal"
	@echo ""
	@echo "  Git initialized. To push to GitHub:"
	@echo "  1. Create repo: gh repo create agente-rs-platform --public"
	@echo "  2. Push:        git remote add origin <url> && git push -u origin main"

# ---- Cloudflare ---------------------------------------------

cf-tunnel-create: ## Create a Cloudflare tunnel (requires cloudflared CLI)
	cloudflared tunnel create agente-rs-public
	@echo "Copy the tunnel ID and credentials to infra/cloudflare/"

cf-tunnel-run: ## Run Cloudflare tunnel locally
	cloudflared tunnel --config infra/cloudflare/tunnel-config.yml run agente-rs-public

# ---- Cleanup ------------------------------------------------

clean: ## Remove build artifacts and temp files
	@find . -name "*.exe" -not -path "./vendor/*" -delete
	@find . -name "tmp" -type d -exec rm -rf {} + 2>/dev/null || true
	@find . -name ".next" -type d -exec rm -rf {} + 2>/dev/null || true
	@echo "Cleaned."

clean-all: ## Remove all generated files including node_modules
	@make clean
	@find . -name "node_modules" -type d -exec rm -rf {} + 2>/dev/null || true
	@echo "Deep clean complete."
