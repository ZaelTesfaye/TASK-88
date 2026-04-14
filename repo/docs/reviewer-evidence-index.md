# Reviewer Evidence Index

Quick-reference guide for reviewers to locate key artifacts in the Multi-Org Data & Media Operations Hub.

## Quick Start

| What | Where |
|---|---|
| README | `README.md` |
| Docker Compose | `docker-compose.yml` |
| Run all tests | `./run_tests.sh` or `./scripts/test.sh` |
| Backend entry point | `backend/cmd/server/main.go` |
| Frontend entry point | `frontend/src/main.js` |

## Entry Points

| Component | File | Purpose |
|---|---|---|
| Server bootstrap | `backend/cmd/server/main.go` | Config load, DB connect, router setup, graceful shutdown |
| Route registration | `backend/internal/router/router.go` | All API routes with middleware chain |
| Frontend app | `frontend/src/main.js` | Vue 3 app creation with Pinia and Router |
| Nginx proxy | `frontend/nginx.conf` | Reverse proxy, security headers, caching |

## Authentication

| Aspect | File |
|---|---|
| JWT + Argon2id service | `backend/internal/auth/auth_service.go` |
| Auth middleware (Bearer, session, timeout) | `backend/internal/auth/auth_middleware.go` |
| Login/logout/refresh handler | `backend/internal/handlers/auth_handler.go` |
| Frontend auth store (idle tracking) | `frontend/src/stores/auth.js` |
| Login page | `frontend/src/pages/LoginPage.vue` |
| Auth unit tests | `backend/tests/unit/auth_test.go` |
| Auth API tests | `backend/tests/api/auth_api_test.go` |
| Frontend auth tests | `frontend/src/tests/unit/auth.store.test.js` |

## Core Modules

| Module | Service File | Handler File | Frontend Page |
|---|---|---|---|
| Org Tree | `backend/internal/org/org_service.go` | `handlers/org_handler.go` | `pages/OrgTreePage.vue` |
| Master Data | `backend/internal/masterdata/master_service.go` | `handlers/master_handler.go` | `pages/MasterDataPage.vue` |
| Versioning | `backend/internal/masterdata/version_service.go` | `handlers/version_handler.go` | `pages/MasterDataPage.vue` |
| Ingestion | `backend/internal/ingestion/job_engine.go` | `handlers/ingestion_handler.go` | `pages/IngestionPage.vue` |
| Connectors | `backend/internal/ingestion/connector.go` | `handlers/integration_handler.go` | -- |
| Playback | `backend/internal/playback/playback_service.go` | `handlers/playback_handler.go` | `pages/PlaybackPage.vue` |
| LRC Parser | `backend/internal/playback/lrc_parser.go` | -- | -- |
| Analytics | `backend/internal/analytics/analytics_service.go` | `handlers/analytics_handler.go` | `pages/AnalyticsPage.vue` |
| Reports | `backend/internal/reports/report_service.go` | `handlers/report_handler.go` | `pages/ReportsPage.vue` |
| Audit | `backend/internal/audit/audit_service.go` | `handlers/audit_handler.go` | -- |
| Security | `backend/internal/security/security_service.go` | `handlers/security_handler.go` | `pages/SecurityAdminPage.vue` |
| Encryption | `backend/internal/security/encryption.go` | -- | -- |
| Integration | `backend/internal/integration/integration_service.go` | `handlers/integration_handler.go` | -- |

## RBAC

| Aspect | File |
|---|---|
| Permission matrix (4 roles, 19 permissions) | `backend/internal/rbac/rbac.go` |
| Object-level scope enforcement | `backend/internal/rbac/scope.go` |
| Frontend route guards | `frontend/src/router/index.js` |
| RBAC unit tests | `backend/tests/unit/rbac_test.go` |
| RBAC API tests | `backend/tests/api/rbac_api_test.go` |

## Security

| Aspect | File |
|---|---|
| AES-256-GCM encryption | `backend/internal/security/encryption.go` |
| Field masking | `backend/internal/security/security_service.go` |
| Key rotation | `backend/internal/security/security_service.go` |
| Egress guard middleware | `backend/internal/middleware/middleware.go` |
| LAN URL validation | `backend/internal/integration/integration_service.go` |
| Encryption tests | `backend/tests/unit/encryption_test.go` |
| Masking tests | `backend/tests/unit/masking_test.go` |

## Configuration

| Aspect | File |
|---|---|
| All env variables | `backend/internal/config/config.go` |
| Docker env defaults | `docker-compose.yml` |
| README config table | `README.md` |

## Models (Database Schema)

| Model Group | File |
|---|---|
| User, Session | `backend/internal/models/user.go`, `session.go` |
| OrgNode, ContextAssignment | `backend/internal/models/org.go` |
| MasterRecord, Version, VersionItem, DeactivationEvent | `backend/internal/models/master.go` |
| ImportSource, IngestionJob, Checkpoint, Failure | `backend/internal/models/ingestion.go` |
| MediaAsset | `backend/internal/models/media.go` |
| KPIDefinition, ReportSchedule, ReportRun | `backend/internal/models/analytics.go` |
| AuditLog, AuditDeleteRequest | `backend/internal/models/audit.go` |
| IntegrationEndpoint, Delivery, ConnectorDefinition | `backend/internal/models/integration.go` |
| SensitiveFieldRegistry, KeyRing, PasswordResetRequest | `backend/internal/models/security.go` |
| RetentionPolicy, LegalHold, PurgeRun | `backend/internal/models/retention.go` |

## Test Suites

| Suite | Location | Runner |
|---|---|---|
| Backend unit tests | `backend/tests/unit/*.go` | `go test ./... -v` |
| Backend API tests | `backend/tests/api/*.go` | `go test ./... -v` |
| Frontend unit tests | `frontend/src/tests/unit/*.test.js` | `npx vitest run` |

## Documentation Files

| Document | Path |
|---|---|
| Architecture | `docs/architecture.md` |
| API Specification | `docs/api-spec.md` |
| RBAC Matrix | `docs/rbac-matrix.md` |
| Security Model | `docs/security-model.md` |
| Requirement Matrix | `docs/requirement-matrix.md` |
| Acceptance Evidence Map | `docs/acceptance-evidence-map.md` |
| Test Coverage Map | `docs/test-coverage-map.md` |
| Mock Disclosure | `docs/mock-and-stub-disclosure.md` |
| Connector Contract | `docs/connector-plugin-contract.md` |
| Retention Policy | `docs/retention-policy.md` |
| TLS Trust Model | `docs/tls-trust-model.md` |
| Password Security | `docs/password-and-account-security.md` |
| Duplicate Detection | `docs/duplicate-detection-rules.md` |
| Playback Formats | `docs/playback-format-compatibility.md` |
| Status Codes | `docs/status-code-policy.md` |
| Network Guard | `docs/offline-network-guard.md` |
| Frontend UX | `docs/frontend-ux-baseline.md` |
