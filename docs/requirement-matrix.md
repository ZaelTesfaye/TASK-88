# Requirement Matrix

Mapping of every functional requirement to its implementing module, endpoint, UI page, test file, and implementation status.

## Authentication and Session Management

| # | Requirement | Module | Endpoint | UI Page | Test File | Status |
|---|---|---|---|---|---|---|
| A1 | JWT authentication with Argon2id | `auth/auth_service.go` | `POST /auth/login` | `LoginPage.vue` | `backend/tests/unit/auth_test.go` | Done |
| A2 | 30-minute idle timeout | `auth/auth_service.go`, `auth/auth_middleware.go` | Enforced on all protected routes | `stores/auth.js` | `backend/tests/unit/auth_test.go::TestSessionIdleTimeout` | Done |
| A3 | 12-hour absolute session timeout | `auth/auth_service.go`, `auth/auth_middleware.go` | Enforced on all protected routes | `stores/auth.js` | `backend/tests/unit/auth_test.go::TestSessionAbsoluteTimeout` | Done |
| A4 | Token refresh | `auth/auth_service.go` | `POST /auth/refresh` | `stores/auth.js` | `backend/tests/api/auth_api_test.go::TestRefreshToken` | Done |
| A5 | Logout/session revocation | `auth/auth_service.go` | `POST /auth/logout` | `stores/auth.js` | `backend/tests/api/auth_api_test.go::TestLogoutRevokesSession` | Done |
| A6 | Frontend idle tracking (30m) | -- | -- | `stores/auth.js` (IDLE_TIMEOUT_MS) | `frontend/src/tests/unit/auth.store.test.js` | Done |
| A7 | Audit login/logout events | `audit/audit_service.go` | Logged in login/logout handlers | -- | `backend/tests/api/auth_api_test.go` | Done |

## Password Policy

| # | Requirement | Module | Endpoint | UI Page | Test File | Status |
|---|---|---|---|---|---|---|
| P1 | Min 12 chars, uppercase, lowercase, digit, symbol | `auth/auth_service.go` | Validated on password set | `LoginPage.vue` | `backend/tests/unit/auth_test.go::TestPasswordComplexity*` | Done |
| P2 | Lockout after 5 failed attempts | `auth/auth_service.go` | `POST /auth/login` | -- | `backend/tests/unit/auth_test.go::TestAccountLockout*` | Done |
| P3 | 15-minute lockout duration | `auth/auth_service.go` | `POST /auth/login` | -- | `backend/tests/api/auth_api_test.go::TestLoginLockedAccount` | Done |
| P4 | Password reset workflow | `security/security_service.go` | `POST /security/password-reset`, `/approve` | `SecurityAdminPage.vue` | `frontend/src/tests/unit/security.test.js` | Done |

## RBAC

| # | Requirement | Module | Endpoint | UI Page | Test File | Status |
|---|---|---|---|---|---|---|
| R1 | 4 roles with permission matrix | `rbac/rbac.go` | All protected routes | `router/index.js` | `backend/tests/unit/rbac_test.go` | Done |
| R2 | Route-level guards | `rbac/rbac.go`, `router/router.go` | All API groups | `router/index.js` | `backend/tests/api/rbac_api_test.go` | Done |
| R3 | Object-level scope (city/dept) | `rbac/scope.go` | Service-layer filtering | -- | `backend/tests/unit/rbac_test.go::TestScopeCheck`, `TestScopeWildcard` | Done |
| R4 | Frontend role-based navigation | -- | -- | `router/index.js` | `frontend/src/tests/unit/router.guards.test.js` | Done |

## Org Tree

| # | Requirement | Module | Endpoint | UI Page | Test File | Status |
|---|---|---|---|---|---|---|
| O1 | Org hierarchy CRUD | `org/org_service.go` | `/api/v1/org/nodes` (CRUD) | `OrgTreePage.vue` | -- | Done |
| O2 | Context switching | `org/org_service.go` | `POST /context/switch` | `stores/context.js` | `frontend/src/tests/unit/context.store.test.js` | Done |
| O3 | Org tree display | `org/org_service.go` | `GET /org/tree` | `OrgTreePage.vue` | -- | Done |

## Master Data

| # | Requirement | Module | Endpoint | UI Page | Test File | Status |
|---|---|---|---|---|---|---|
| M1 | Entity CRUD (7 types) | `masterdata/master_service.go` | `/api/v1/master/:entity` | `MasterDataPage.vue` | `frontend/src/tests/unit/master-data.test.js` | Done |
| M2 | Entity-specific validation | `masterdata/master_service.go` | POST/PUT endpoints | -- | `backend/tests/unit/ingestion_test.go::TestValidationRule*` | Done |
| M3 | Deactivation with reason | `masterdata/master_service.go` | `POST /:entity/:id/deactivate` | `MasterDataPage.vue` | -- | Done |
| M4 | Duplicate detection | `masterdata/master_service.go` | Create endpoint (409 on dup) | -- | -- | Done |
| M5 | CSV/XLSX import | `masterdata/master_service.go` | Import via master handler | `MasterDataPage.vue` | `backend/tests/unit/ingestion_test.go::TestCSVParsing` | Done |
| M6 | Import file validation (50MB, UTF-8) | `ingestion/validation.go` | Import handler | -- | `backend/tests/unit/ingestion_test.go::TestImportFileSizeLimit` | Done |

## Versioning

| # | Requirement | Module | Endpoint | UI Page | Test File | Status |
|---|---|---|---|---|---|---|
| V1 | Draft/review/activate lifecycle | `masterdata/version_service.go` | `/api/v1/versions/:entity` | `MasterDataPage.vue` | -- | Done |
| V2 | Reviewer != creator | `masterdata/version_service.go` | `POST /:entity/:id/review` | -- | -- | Done |
| V3 | SystemAdmin-only activation | `masterdata/version_service.go`, `router/router.go` | `POST /:entity/:id/activate` | -- | -- | Done |
| V4 | Rollback to archived version | `masterdata/version_service.go` | Activate archived version | -- | -- | Done |
| V5 | Version diff | `masterdata/version_service.go` | `GET /:entity/:id/diff` | -- | -- | Done |

## Ingestion

| # | Requirement | Module | Endpoint | UI Page | Test File | Status |
|---|---|---|---|---|---|---|
| I1 | Import sources CRUD | `ingestion/job_engine.go` | `/api/v1/ingestion/sources` | `IngestionPage.vue` | -- | Done |
| I2 | Priority-based job queue | `ingestion/job_engine.go` | `/api/v1/ingestion/jobs` | `IngestionPage.vue` | `backend/tests/unit/ingestion_test.go` | Done |
| I3 | Exponential backoff retry | `ingestion/job_engine.go` | `POST /jobs/:id/retry` | -- | `backend/tests/unit/ingestion_test.go::TestExponentialBackoff` | Done |
| I4 | Checkpoint every 1000 records | `ingestion/job_engine.go` | -- | -- | `backend/tests/unit/ingestion_test.go::TestCheckpointWrite` | Done |
| I5 | Failed job acknowledgment | `ingestion/job_engine.go` | `POST /jobs/:id/acknowledge` | `IngestionPage.vue` | `backend/tests/unit/ingestion_test.go::TestRetryLimit` | Done |
| I6 | Missed run catch-up | `ingestion/scheduler.go` | On startup | -- | -- | Done |
| I7 | Starvation prevention (+10 after 30m) | `ingestion/job_engine.go` | ProcessNextJob() | -- | -- | Done |
| I8 | Dependency groups | `ingestion/job_engine.go` | CreateJob with dependency_group | -- | `backend/tests/unit/ingestion_test.go::TestJobStateTransitions` | Done |
| I9 | Connector plugin interface | `ingestion/connector.go` | `/api/v1/integrations/connectors` | -- | -- | Done |

## Playback

| # | Requirement | Module | Endpoint | UI Page | Test File | Status |
|---|---|---|---|---|---|---|
| PB1 | Media CRUD | `playback/playback_service.go` | `/api/v1/media` | `PlaybackPage.vue` | `frontend/src/tests/unit/playback.test.js` | Done |
| PB2 | LRC parsing (line + word level) | `playback/lrc_parser.go` | `POST /media/:id/lyrics/parse` | `PlaybackPage.vue` | `backend/tests/unit/lrc_parser_test.go` | Done |
| PB3 | Lyrics search | `playback/lrc_parser.go` | `GET /media/:id/lyrics/search` | `PlaybackPage.vue` | `backend/tests/unit/lrc_parser_test.go::TestSearchLyrics` | Done |
| PB4 | Supported formats (mp3/wav/flac/m4a) | `playback/playback_service.go` | `GET /media/formats/supported` | -- | -- | Done |
| PB5 | UTF-16 LRC handling | `playback/lrc_parser.go` | -- | -- | `backend/tests/unit/lrc_parser_test.go` | Done |
| PB6 | Audio streaming | `playback/playback_service.go` | `GET /media/:id/stream` | `PlaybackPage.vue` | -- | Done |

## Analytics

| # | Requirement | Module | Endpoint | UI Page | Test File | Status |
|---|---|---|---|---|---|---|
| AN1 | KPI definitions CRUD | `analytics/analytics_service.go` | `/api/v1/analytics/kpis/definitions` | `AnalyticsPage.vue` | -- | Done |
| AN2 | KPI computation with trends | `analytics/analytics_service.go` | `GET /analytics/kpis` | `AnalyticsPage.vue` | -- | Done |
| AN3 | Time-series trend data | `analytics/analytics_service.go` | `GET /analytics/trends` | `AnalyticsPage.vue` | -- | Done |

## Reports

| # | Requirement | Module | Endpoint | UI Page | Test File | Status |
|---|---|---|---|---|---|---|
| RP1 | Schedule CRUD | `reports/report_service.go` | `/api/v1/reports/schedules` | `ReportsPage.vue` | `frontend/src/tests/unit/reports.test.js` | Done |
| RP2 | Run history + download | `reports/report_service.go` | `/api/v1/reports/runs` | `ReportsPage.vue` | -- | Done |
| RP3 | Access check | `reports/report_service.go` | `GET /runs/:id/access-check` | -- | -- | Done |
| RP4 | Manual trigger | `reports/report_service.go` | `POST /schedules/:id/trigger` | `ReportsPage.vue` | -- | Done |

## Security Admin

| # | Requirement | Module | Endpoint | UI Page | Test File | Status |
|---|---|---|---|---|---|---|
| S1 | Sensitive field registry | `security/security_service.go` | `/api/v1/security/sensitive-fields` | `SecurityAdminPage.vue` | `backend/tests/unit/masking_test.go` | Done |
| S2 | Field masking (last4/email/phone/full) | `security/security_service.go` | Applied in service layer | -- | `backend/tests/unit/masking_test.go` | Done |
| S3 | Key ring + rotation | `security/security_service.go` | `/api/v1/security/keys` | `SecurityAdminPage.vue` | -- | Done |
| S4 | AES-256-GCM encryption | `security/encryption.go` | Used by biometric feature | -- | `backend/tests/unit/encryption_test.go` | Done |
| S5 | Retention policies | `security/security_service.go` | `/api/v1/security/retention-policies` | `SecurityAdminPage.vue` | `backend/tests/unit/retention_test.go` | Done |
| S6 | Legal holds | `security/security_service.go` | `/api/v1/security/legal-holds` | `SecurityAdminPage.vue` | `backend/tests/unit/retention_test.go` | Done |
| S7 | Purge dry-run + execute | `security/security_service.go` | `/api/v1/security/purge-runs` | `SecurityAdminPage.vue` | `backend/tests/unit/retention_test.go` | Done |

## Audit

| # | Requirement | Module | Endpoint | UI Page | Test File | Status |
|---|---|---|---|---|---|---|
| AU1 | Append-only audit logging | `audit/audit_service.go` | `/api/v1/audit/logs` | -- | -- | Done |
| AU2 | Dual-approval deletion | `audit/audit_service.go` | `/api/v1/audit/delete-requests` | -- | -- | Done |

## Integration

| # | Requirement | Module | Endpoint | UI Page | Test File | Status |
|---|---|---|---|---|---|---|
| IN1 | Webhook endpoints CRUD | `integration/integration_service.go` | `/api/v1/integrations/endpoints` | -- | -- | Done |
| IN2 | At-least-once delivery | `integration/integration_service.go` | `SendEvent()` | -- | -- | Done |
| IN3 | LAN-only URL validation | `integration/integration_service.go` | `ValidateEndpointURL()` | -- | -- | Done |
| IN4 | Delivery retry with backoff | `integration/integration_service.go` | `markDeliveryFailed()` | -- | -- | Done |
| IN5 | Connector definitions CRUD | `integration/integration_service.go` | `/api/v1/integrations/connectors` | -- | -- | Done |

## Infrastructure

| # | Requirement | Module | Endpoint | UI Page | Test File | Status |
|---|---|---|---|---|---|---|
| IF1 | Docker Compose orchestration | `docker-compose.yml` | -- | -- | -- | Done |
| IF2 | Health check endpoint | `router/router.go` | `GET /health` | -- | -- | Done |
| IF3 | Structured JSON logging | `logging/logger.go` | -- | -- | -- | Done |
| IF4 | Correlation ID propagation | `middleware/middleware.go` | All requests | -- | `backend/tests/api/auth_api_test.go::TestLoginErrorIncludesCorrelationId` | Done |
| IF5 | Egress guard (LAN-only) | `middleware/middleware.go` | All requests | -- | -- | Done |
| IF6 | Graceful shutdown | `cmd/server/main.go` | -- | -- | -- | Done |
