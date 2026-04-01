SHELL := /usr/bin/env bash

.DEFAULT_GOAL := help

.PHONY: help env install backend-install frontend-install db-create db-drop db-reset schema backend-run frontend-run dev backend-fmt backend-vet backend-revive backend-lint lint backend-test frontend-test backend-docs frontend-build backend-build test build clean

help: ## Show this help message
	@printf "\n"
	@printf "\033[1;36m"
	@printf " ░▒▓███████▓▒░▒▓██████████████▓▒░ ░▒▓██████▓▒░░▒▓███████▓▒░▒▓████████▓▒░ \n"
	@printf "░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ ░▒▓█▓▒░     \n"
	@printf "░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ ░▒▓█▓▒░     \n"
	@printf " ░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓████████▓▒░▒▓███████▓▒░  ░▒▓█▓▒░     \n"
	@printf "       ░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ ░▒▓█▓▒░     \n"
	@printf "       ░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ ░▒▓█▓▒░     \n"
	@printf "░▒▓███████▓▒░░▒▓█▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ ░▒▓█▓▒░     \n"
	@printf "\033[0m\n"

	@printf "\033[1;37m                                                   Inventory Core System\033[0m\n\n"

	@awk 'BEGIN { FS = ":.*## " } \
	/^#@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 4) } \
	/^[a-zA-Z0-9_.-]+:.*## / { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

#@ Setup
env: ## Create local env files if missing
	@test -f backend/.env || cp backend/.env.example backend/.env
	@test -f frontend/.env || cp frontend/.env.example frontend/.env
	@printf "Environment files are ready.\n"

install: backend-install frontend-install ## Install backend and frontend dependencies

backend-install: ## Refresh backend dependencies
	@cd backend && go mod tidy

frontend-install: ## Install frontend dependencies
	@cd frontend && npm install

#@ Database
db-create: env ## Create the PostgreSQL database from backend/.env if missing
	@set -euo pipefail; \
	set -a; source backend/.env; set +a; \
	db_name=$${DATABASE_URL##*/}; \
	db_name=$${db_name%%\?*}; \
	admin_url=$${DATABASE_URL%/*}/postgres; \
	if [[ "$$DATABASE_URL" == *\?* ]]; then admin_url="$$admin_url?$${DATABASE_URL#*\?}"; fi; \
	exists=$$(psql "$$admin_url" -tAc "SELECT 1 FROM pg_database WHERE datname = '$$db_name'" | tr -d '[:space:]'); \
	if [[ "$$exists" != "1" ]]; then \
		printf "Creating PostgreSQL database %s...\n" "$$db_name"; \
		psql "$$admin_url" -c "CREATE DATABASE \"$$db_name\"" >/dev/null || { \
			printf "Failed to create database %s. Ensure the role in backend/.env can create databases, or create it once as postgres.\n" "$$db_name"; \
			exit 1; \
		}; \
	else \
		printf "PostgreSQL database %s already exists.\n" "$$db_name"; \
	fi

db-drop: env ## Drop the PostgreSQL database from backend/.env if it exists
	@set -euo pipefail; \
	set -a; source backend/.env; set +a; \
	db_name=$${DATABASE_URL##*/}; \
	db_name=$${db_name%%\?*}; \
	admin_url=$${DATABASE_URL%/*}/postgres; \
	if [[ "$$DATABASE_URL" == *\?* ]]; then admin_url="$$admin_url?$${DATABASE_URL#*\?}"; fi; \
	if [[ "$$db_name" == "postgres" ]]; then \
		printf "Refusing to drop the postgres maintenance database.\n"; \
		exit 1; \
	fi; \
	exists=$$(psql "$$admin_url" -tAc "SELECT 1 FROM pg_database WHERE datname = '$$db_name'" | tr -d '[:space:]'); \
	if [[ "$$exists" != "1" ]]; then \
		printf "PostgreSQL database %s does not exist.\n" "$$db_name"; \
	else \
		printf "Dropping PostgreSQL database %s...\n" "$$db_name"; \
		psql "$$admin_url" -c "DROP DATABASE \"$$db_name\" WITH (FORCE)" >/dev/null || { \
			printf "Failed to drop database %s. Ensure the role in backend/.env owns the database or can drop databases, or drop it once as postgres.\n" "$$db_name"; \
			exit 1; \
		}; \
		printf "PostgreSQL database %s dropped.\n" "$$db_name"; \
	fi

db-reset: ## Drop and recreate the PostgreSQL database from backend/.env, then apply schema
	@$(MAKE) db-drop
	@$(MAKE) schema

schema: db-create ## Create the database if needed, then apply the PostgreSQL schema using backend/.env
	@set -a; source backend/.env; set +a; \
	psql -v ON_ERROR_STOP=1 "$$DATABASE_URL" -f backend/internal/platform/postgres/schema.sql

#@ Run
backend-run: env ## Run the Go API with backend/.env loaded
	@set -a; source backend/.env; set +a; \
	cd backend && go run ./cmd/api

frontend-run: env ## Run the Vite frontend
	@cd frontend && npm run dev

dev: env ## Run backend and frontend together
	@set -a; source backend/.env; set +a; \
	cd backend && go run ./cmd/api & \
	BACKEND_PID=$$!; \
	trap 'kill $$BACKEND_PID' EXIT INT TERM; \
	cd frontend && npm run dev

#@ Quality
backend-fmt: ## Check backend Go formatting
	@printf "Checking backend Go formatting...\n"
	@cd backend && files=$$(find . -type f -name '*.go' -not -path './docs/swagger/*'); \
	needs_fmt=$$(gofmt -l $$files); \
	if [[ -n "$$needs_fmt" ]]; then \
		printf "Go files need formatting:\n%s\n" "$$needs_fmt"; \
		exit 1; \
	fi
	@printf "Backend Go formatting check passed.\n"

backend-vet: ## Run go vet on the backend
	@printf "Running go vet on backend...\n"
	@cd backend && go vet ./...
	@printf "go vet passed.\n"

backend-revive: ## Run revive on the backend using backend/revive.toml
	@printf "Running revive on backend...\n"
	@cd backend && go run github.com/mgechev/revive@v1.7.0 -config ./revive.toml ./...
	@printf "revive passed.\n"

backend-lint: backend-fmt backend-vet backend-revive ## Run backend formatting and static analysis checks
	@printf "Backend lint checks passed.\n"

lint: backend-lint ## Run lint checks
	@printf "All lint checks passed.\n"

backend-test: ## Run backend tests
	@cd backend && go test ./...

backend-docs: ## Regenerate backend Swagger docs from Go annotations
	@cd backend && go run github.com/swaggo/swag/cmd/swag@v1.16.6 init \
		-g main.go \
		-d ./cmd/api,./internal/app/http,./internal/domain \
		--parseInternal \
		-o ./docs/swagger \
		--outputTypes go,json,yaml

frontend-test: ## Run frontend tests
	@cd frontend && npm run test

test: backend-test frontend-test ## Run backend and frontend tests

#@ Build

frontend-build: ## Build the frontend production bundle
	@cd frontend && npm run build

backend-build: ## Build the backend binary
	@mkdir -p backend/bin
	@cd backend && go build -o ./bin/api ./cmd/api

build: backend-build frontend-build ## Build backend binary and frontend bundle

#@ Utilities

clean: ## Remove generated build and test output
	@rm -rf backend/bin frontend/dist frontend/coverage frontend/.vitest frontend/vite.config.js frontend/vite.config.d.ts frontend/*.tsbuildinfo