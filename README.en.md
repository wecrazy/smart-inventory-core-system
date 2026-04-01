# Smart Inventory Core System

Assessment implementation for a stock management system with auditable inventory changes, reservation-safe outbound flows, and a React operator UI.

The primary submission document is [README.md](README.md) in Indonesian to match the assessment requirement. This file is an English companion for convenience.

## Overview

### Stack

| Layer | Technology |
| --- | --- |
| Backend | Go 1.26, Fiber v3, pgx, PostgreSQL |
| Frontend | React 18, Vite, Redux Toolkit, RTK Query, React Router |
| Tooling | Make, Vitest, Testing Library |

### What It Does

- Tracks inventory by SKU, name, and customer.
- Separates physical stock, reserved stock, and derived available stock.
- Models stock adjustment as an auditable transaction instead of a silent overwrite.
- Supports stock-in transitions: `CREATED -> IN_PROGRESS -> DONE` with cancellation before completion.
- Supports stock-out reservation flow: `ALLOCATED -> IN_PROGRESS -> DONE` with rollback on cancellation.
- Exposes report views for completed stock movements only.

## Repository Layout

```text
backend/
  cmd/api/
  docs/swagger/
  internal/app/
  internal/domain/
  internal/platform/
frontend/
AI_NOTES.md
README.md
README.en.md
```

## Architecture

The backend uses a layered structure:

- `internal/domain` contains business entities and status rules.
- `internal/app/service` contains use-case orchestration and validation.
- `internal/platform/postgres` contains PostgreSQL repositories, transactions, and row locking.
- `internal/app/http` contains HTTP handlers, routing, and Swagger wiring.

The frontend uses Redux Toolkit and RTK Query:

- `frontend/src/app/store.ts` configures the global store.
- `frontend/src/app/baseApi.ts` and split endpoint modules handle API communication.
- pages are separated by workflow: Inventory, Stock In, Stock Out, and Reports.

The Mermaid diagrams below are there to clarify the two hardest parts of the system at a glance: the layered architecture and the transaction status workflows. On GitHub these render as static diagrams, which is the right fit here; this README needs clarity, not animation.

### Architecture Diagram

```mermaid
flowchart LR
  subgraph FE[Frontend React]
    Pages[Pages\nInventory | Stock In | Stock Out | Reports]
    State[Redux Store + RTK Query]
    Pages --> State
  end

  subgraph BE[Backend Go Fiber]
    HTTP[HTTP Router + Handlers]
    Service[Service Layer\nvalidation + use-case orchestration]
    Domain[Domain Rules\nstatus transitions + audit rules]
    Repo[PostgreSQL Repository\ndatabase transactions + row locking]
    HTTP --> Service
    Service --> Domain
    Service --> Repo
  end

  DB[(PostgreSQL)]
  Swagger[Swagger UI]

  State --> HTTP
  Repo --> DB
  HTTP --> Swagger
```

### Workflow Diagrams

#### Stock-In

```mermaid
stateDiagram-v2
  [*] --> CREATED
  CREATED --> IN_PROGRESS: operator starts processing
  CREATED --> CANCELLED: cancelled before completion
  IN_PROGRESS --> DONE: stock applied to inventory
  IN_PROGRESS --> CANCELLED: cancelled before completion
  DONE --> [*]
  CANCELLED --> [*]

  note right of DONE
    physical_stock increases
    included in done-only reporting
  end note
```

#### Stock-Out

```mermaid
stateDiagram-v2
  [*] --> ALLOCATED: reserve available stock
  ALLOCATED --> IN_PROGRESS: operator starts picking
  ALLOCATED --> CANCELLED: release reservation
  IN_PROGRESS --> DONE: finalize outbound movement
  IN_PROGRESS --> CANCELLED: release reservation
  DONE --> [*]
  CANCELLED --> [*]

  note right of ALLOCATED
    reserved_stock increases
    available_stock decreases
  end note

  note right of DONE
    physical_stock decreases
    included in done-only reporting
  end note
```

`ADJUSTMENT` is still stored as an auditable transaction, but it is intentionally excluded from reports because reporting only includes `DONE` stock-in and stock-out transactions.

## Architecture Decision Summary

### Context

The assessment requires strict transaction transitions, done-only reporting, and synchronized backend/frontend workflows. The highest-risk technical area is stock-out allocation, because reserved stock must remain safe under concurrent requests.

### Design Decisions

- The backend API uses Go Fiber v3.
- PostgreSQL access is implemented with `pgx`.
- The repository layer performs stock mutations inside database transactions with row locking.
- The frontend uses React with Redux Toolkit and RTK Query.
- The repository is structured as a monorepo with `backend/` and `frontend/`.

Inventory is modeled with `physical_stock`, `reserved_stock`, and `available_stock = physical_stock - reserved_stock`.

Transactions are modeled as `STOCK_IN`, `STOCK_OUT`, and `ADJUSTMENT`, while status history is stored separately so the audit trail remains explicit.

### Consequences

Main benefits:

- stock-out allocation and cancellation rollback stay auditable and concurrency-aware
- reporting remains simple because only `DONE` stock-in and stock-out transactions are reportable
- the frontend can bind directly to focused workflow endpoints

Main tradeoffs:

- database bootstrap still relies on a schema SQL file instead of versioned migrations
- `customer_name` is still stored directly on inventory items
- repository-level concurrency tests are still missing

## Frontend Summary

### Purpose

The frontend is an operator UI for inventory visibility, inventory creation, stock adjustment, stock-in execution, stock-out execution, and done-only reporting.

### Frontend Routes

- `/` for inventory search, inventory creation, and stock adjustment
- `/stock-in` for inbound transaction creation and status progression
- `/stock-out` for outbound allocation, reservation rollback, and completion
- `/reports` for completed transaction reports only

### Frontend Data Flow

- RTK Query is the single HTTP client layer
- endpoint modules are split by domain under `frontend/src/app/`
- the UI expects the backend envelope shape `{ data, error? }`

### Backend Dependency Used By The Frontend

The frontend relies on these endpoints:

- `GET /api/v1/inventory`
- `POST /api/v1/inventory`
- `POST /api/v1/inventory/adjustments`
- `POST /api/v1/stock-in`
- `GET /api/v1/stock-in`
- `PATCH /api/v1/stock-in/:id/status`
- `POST /api/v1/stock-in/:id/cancel`
- `POST /api/v1/stock-out`
- `GET /api/v1/stock-out`
- `PATCH /api/v1/stock-out/:id/status`
- `POST /api/v1/stock-out/:id/cancel`
- `GET /api/v1/reports`
- `GET /api/v1/reports/export`

### Frontend Testing Status

- current frontend tests cover the app shell and a small API utility
- deeper coverage for form submission and mutation feedback is still a good next step

## Quick Start

From the repository root:

```bash
git clone https://github.com/wecrazy/smart-inventory-core-system.git
cd smart-inventory-core-system
make env
make install
make schema
make dev
```

Clone the repository from:

- `https://github.com/wecrazy/smart-inventory-core-system.git`

Application URLs:

- Frontend: `http://localhost:5173`
- Backend API: `http://localhost:8080/api/v1`
- Backend Swagger UI: `http://localhost:8080/swagger/index.html`

After `make dev`, follow the step-by-step operator flow in [TESTING_GUIDE.md](TESTING_GUIDE.md). An English version is also available in [TESTING_GUIDE.en.md](TESTING_GUIDE.en.md).

## Make Targets

| Command | Purpose |
| --- | --- |
| `make env` | Create local env files if missing |
| `make install` | Install backend and frontend dependencies |
| `make db-create` | Create the configured PostgreSQL database if it does not exist |
| `make db-drop` | Drop the configured PostgreSQL database if it exists |
| `make db-reset` | Drop, recreate, and reapply the configured PostgreSQL schema |
| `make schema` | Create the database if needed, then apply the PostgreSQL schema |
| `make backend-run` | Run the Go API |
| `make frontend-run` | Run the Vite frontend |
| `make dev` | Run backend and frontend together |
| `make backend-fmt` | Check whether backend Go files are formatted |
| `make backend-vet` | Run `go vet` on the backend |
| `make backend-revive` | Run `revive` using `backend/revive.toml` |
| `make backend-lint` | Run backend format checks, `go vet`, and `revive` |
| `make lint` | Alias for backend lint checks |
| `make test` | Run backend tests and frontend tests |
| `make backend-docs` | Regenerate Swagger docs from Go annotations |
| `make build` | Build backend binary and frontend bundle |
| `make clean` | Remove generated artifacts |

## Environment Defaults

```env
APP_ENV=development
HTTP_PORT=8080
DATABASE_URL=postgres://wegil:postgres@localhost:5432/smart_inventory?sslmode=disable
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

## Verification

Verified commands:

- `cd backend && go test ./...`
- `make backend-docs`
- `cd frontend && npm run test`
- `cd frontend && npm run build`
- `make test`
- `make build`
- `make db-create && make schema`

## API Documentation

- backend API documentation is generated from handler annotations and served at `http://localhost:8080/swagger/index.html`

## Current Tradeoffs

- Database bootstrapping still uses a schema SQL file instead of versioned migrations.
- Frontend coverage is still smoke-level, not deep feature-level.
- Backend concurrency behavior uses row locks, but repository-level concurrency tests are still missing.
