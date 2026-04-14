# Multi-Org Data & Media Operations Hub

A full-stack platform for managing multi-organisation data operations, media assets, integrations, and reporting. Built with Vue 3 on the frontend, Go (Gin) on the backend, and MySQL 8 for persistence.

## Architecture Overview

```
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│   Frontend   │──────▶│   Backend   │──────▶│    MySQL    │
│  Vue 3 SPA   │ :3000 │  Go / Gin   │ :8080 │     8.0     │ :3306
│  + Nginx     │       │  REST API   │       │             │
└─────────────┘       └─────────────┘       └─────────────┘
```

- **Frontend** -- Vue 3 + Vite single-page application served by Nginx. Proxies `/api/*` to the backend.
- **Backend** -- Go REST API using the Gin framework. Handles authentication, RBAC, data operations, media management, scheduling, and integrations.
- **Database** -- MySQL 8.0 with 27 tables covering users, org hierarchy, master data, ingestion pipelines, media, analytics, audit, security, retention, and connectors.

## Prerequisites

| Tool            | Minimum Version |
|-----------------|-----------------|
| Docker          | 20.10+          |
| Docker Compose  | 2.0+            |
| Go (local dev)  | 1.23+           |
| Node.js (local) | 20 LTS          |

## Quick Start

```bash
# Clone the repository and start all services
docker-compose up --build

# Or use the helper script
./scripts/dev-up.sh
```

Docker Compose will:
1. Start MySQL 8.0 and run the init migration automatically.
2. Build and start the Go backend (waits for MySQL health check).
3. Build and start the Vue frontend with Nginx reverse proxy.

## Service Endpoints

| Service  | URL                        | Description                 |
|----------|----------------------------|-----------------------------|
| Frontend | http://localhost:3000      | Vue 3 SPA                  |
| Backend  | http://localhost:8080      | Go REST API                |
| MySQL    | localhost:3306             | Database (hub_db)           |

## Default Credentials

| Field    | Value              |
|----------|--------------------|
| Username | `admin`            |
| Password | `Admin@12345678`   |
| Role     | `system_admin`     |

> Change the default password immediately after first login.

## Configuration

All backend configuration is passed through environment variables. These are defined in `docker-compose.yml` and can be overridden with a `.env` file.

### Database

| Variable      | Default         | Description                          |
|---------------|-----------------|--------------------------------------|
| `DB_HOST`     | `mysql`         | MySQL hostname                       |
| `DB_PORT`     | `3306`          | MySQL port                           |
| `DB_USER`     | `hub_user`      | MySQL username                       |
| `DB_PASSWORD` | `hub_password`  | MySQL password                       |
| `DB_NAME`     | `hub_db`        | MySQL database name                  |

### Authentication & Security

| Variable              | Default                                          | Description                                      |
|-----------------------|--------------------------------------------------|--------------------------------------------------|
| `JWT_SECRET`          | `change-me-in-production-use-strong-secret-here` | HMAC secret for signing JWTs                     |
| `JWT_ISSUER`          | `multi-org-hub`                                  | Issuer claim in JWTs                             |
| `ENABLE_TLS`          | `false`                                          | Enable HTTPS on the backend                      |
| `ENABLE_BIOMETRIC`    | `false`                                          | Enable biometric authentication support          |
| `CORS_ORIGINS`        | `http://localhost:3000`                           | Comma-separated allowed CORS origins (no wildcards with credentials) |
| `ALLOWED_HOSTS`       | RFC 1918 ranges + loopback                       | Comma-separated CIDR ranges allowed to connect   |
| `KEY_ROTATION_DAYS`   | `90`                                             | Days between automatic encryption key rotations  |

### Application

| Variable                    | Default          | Description                                               |
|-----------------------------|------------------|-----------------------------------------------------------|
| `APP_PORT`                  | `8080`           | HTTP listen port                                          |
| `APP_TIMEZONE`              | `America/New_York` | Server timezone for scheduling                          |
| `APP_ENV`                   | `production`     | Environment (`production`, `staging`, `development`)      |
| `LOG_LEVEL`                 | `info`           | Log level (`debug`, `info`, `warn`, `error`)              |
| `RETENTION_PURGE_DAYS`      | `30`             | Frequency in days for retention purge job                 |
| `MISSED_RUN_POLICY`         | `catch-up-once`  | How missed scheduled runs are handled                     |
| `DIR_SYNC_INTERVAL_MINUTES` | `15`             | Interval for directory sync operations                    |

## Local Development Without Docker

### Backend

```bash
cd backend
go mod download

# Set environment variables (or create a .env file)
export DB_HOST=127.0.0.1 DB_PORT=3306 DB_USER=hub_user DB_PASSWORD=hub_password DB_NAME=hub_db
export JWT_SECRET=dev-secret APP_PORT=8080

go run ./cmd/server
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

The Vite dev server starts on `http://localhost:5173` by default. Configure the API proxy in `vite.config.js` to point at the backend.

## Running Tests

```bash
# Run all tests (backend + frontend)
./run_tests.sh

# Or via the wrapper script
./scripts/test.sh

# Backend only
cd backend && go test ./... -v -count=1

# Frontend only
cd frontend && npx vitest run
```

## Linting

```bash
./scripts/lint.sh
```

This runs `golangci-lint` (or `go vet` as fallback) for the backend and ESLint for the frontend.

## Database Migrations

The initial schema is applied automatically when the MySQL container starts for the first time via Docker entrypoint. The migration file is located at:

```
backend/migrations/init.sql
```

For subsequent migrations:

1. Add new `.sql` files to `backend/migrations/` with a sequential prefix (e.g., `002_add_notifications.sql`).
2. Apply manually or integrate with a migration tool like `golang-migrate` or `goose`.

## Project Structure

```
repo/
├── docker-compose.yml          # Orchestrates all services
├── run_tests.sh                # Global test runner
├── .gitignore
├── README.md
│
├── backend/
│   ├── Dockerfile              # Multi-stage Go build
│   ├── go.mod
│   ├── go.sum
│   ├── cmd/
│   │   └── server/             # Application entry point
│   ├── internal/               # Private application packages
│   └── migrations/
│       └── init.sql            # Full database schema + seed data
│
├── frontend/
│   ├── Dockerfile              # Multi-stage Node build + Nginx
│   ├── nginx.conf              # Nginx reverse proxy config
│   ├── package.json
│   ├── vite.config.js
│   ├── vitest.config.js
│   ├── index.html
│   └── src/                    # Vue 3 application source
│
└── scripts/
    ├── dev-up.sh               # Start development environment
    ├── lint.sh                 # Run linters for both stacks
    └── test.sh                 # Test wrapper
```

## API Overview

All API endpoints are prefixed with `/api/v1/`. Authentication uses JWT bearer tokens.

### Endpoint Groups

| Group                | Prefix                        | Description                                     |
|----------------------|-------------------------------|-------------------------------------------------|
| Auth                 | `/api/v1/auth`                | Login, logout, token refresh                    |
| Org Nodes            | `/api/v1/org`                 | Organisation tree CRUD, hierarchy navigation    |
| Context              | `/api/v1/context`             | Org context switching and current context       |
| Master Records       | `/api/v1/master`              | Master data CRUD, deactivation                  |
| Versions             | `/api/v1/versions`            | Version drafts, review, activation              |
| Media                | `/api/v1/media`               | Upload, streaming, lyrics, cover art            |
| Ingestion            | `/api/v1/ingestion`           | Import sources, job management, failure review  |
| Analytics            | `/api/v1/analytics`           | KPI definitions, trends, dashboard data         |
| Reports              | `/api/v1/reports`             | Schedule management, run history, downloads     |
| Integrations         | `/api/v1/integrations`        | Endpoint CRUD, delivery log, connectors         |
| Audit                | `/api/v1/audit`               | Audit log queries, dual-approval deletion       |
| Security             | `/api/v1/security`            | Sensitive fields, key ring, retention, purge    |

## Security Features

- **JWT Authentication** -- Stateless token-based auth with configurable expiry and refresh tokens.
- **Role-Based Access Control (RBAC)** -- Four-tier role system (system_admin, data_steward, operations_analyst, standard_user) with city/department scope enforcement.
- **Multi-Factor Authentication** -- Optional TOTP-based MFA per user account.
- **Field-Level Encryption** -- Sensitive fields tracked in a registry with configurable masking strategies; encryption keys managed via the key ring with automatic rotation.
- **Immutable Audit Trail** -- All significant actions logged with old/new value diffs; deletion requires formal approval.
- **Data Retention Policies** -- Configurable per-entity retention with legal hold support to suspend purges.
- **Network Restrictions** -- Configurable allowed host CIDR ranges.
- **Password Security** -- Argon2id hashing, reset token expiry, account lockout support.
- **Input Validation** -- Request validation and sanitisation at the API layer.
- **Security Headers** -- X-Frame-Options, X-Content-Type-Options, XSS protection, and referrer policy enforced by Nginx.
