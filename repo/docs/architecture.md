# Architecture

## Technology Stack

| Layer | Technology | Version | Purpose |
|---|---|---|---|
| Frontend | Vue 3 + Vite | Vue 3, Vite 5 | Single-page application |
| State Management | Pinia | 2.x | Reactive stores |
| Routing | Vue Router | 4.x | Client-side navigation with role guards |
| Charts | ECharts | 5.x | Analytics visualizations |
| HTTP Client | Axios | 1.x | API communication with interceptors |
| UI Framework | SCSS (custom) | -- | Design system with 8px grid, color tokens |
| Backend | Go + Gin | Go 1.23, Gin | REST API framework |
| ORM | GORM | 2.x | Database abstraction + auto-migration |
| Database | MySQL | 8.0 | Relational persistence (27 tables) |
| Auth | JWT + Argon2id | golang-jwt/v5 | Stateless auth with session tracking |
| Encryption | AES-256-GCM | crypto/aes | Field-level encryption for biometric data |
| Scheduling | robfig/cron | v3 | Ingestion and report scheduling |
| Logging | zap | 1.x | Structured JSON logging with redaction |
| File Parsing | excelize | v2 | XLSX file reading for imports |
| PDF Generation | jung-kurt/gofpdf | v1 | PDF report output |
| Containerization | Docker + Docker Compose | -- | Three-service orchestration |

## System Architecture Diagram

```
+================================================================+
|                      DEPLOYMENT TOPOLOGY                        |
|                     (docker-compose.yml)                        |
+================================================================+

  +--------------------+    +--------------------+    +------------------+
  |   hub-frontend     |    |   hub-backend      |    |   hub-mysql      |
  |   (Nginx + Vue)    |    |   (Go/Gin)         |    |   (MySQL 8.0)    |
  |   :3000 -> :80     |    |   :8080            |    |   :3306          |
  +--------------------+    +--------------------+    +------------------+
  | Dockerfile:        |    | Dockerfile:        |    | image: mysql:8.0 |
  |  multi-stage build |    |  multi-stage build |    | volumes:         |
  |  Node -> Nginx     |    |  Go build -> alpine|    |  mysql_data      |
  | volumes: (none)    |    | volumes:           |    |  init.sql (DDL)  |
  | depends_on:        |    |  media_data        |    | healthcheck:     |
  |  backend           |    |  export_data       |    |  mysqladmin ping |
  +--------------------+    |  import_data       |    +------------------+
                            | depends_on:        |
                            |  mysql (healthy)   |
                            +--------------------+

  Shared Docker volumes: mysql_data, media_data, export_data, import_data
```

## Module Boundaries

```
repo/
 +-- backend/
 |    +-- cmd/server/main.go            # Entry point, HTTP server lifecycle
 |    +-- internal/
 |    |    +-- config/config.go         # Environment-based configuration singleton
 |    |    +-- database/database.go     # GORM connection, pooling, auto-migration
 |    |    +-- router/router.go         # Route registration and middleware wiring
 |    |    +-- middleware/middleware.go  # RequestID, EgressGuard, Recovery, Logging
 |    |    +-- errors/errors.go         # Standardized AppError type and helpers
 |    |    +-- logging/logger.go        # zap-based structured logging with redaction
 |    |    +-- auth/                    # JWT generation, validation, session mgmt
 |    |    |    +-- auth_service.go     # Argon2id hashing, token pair, lockout
 |    |    |    +-- auth_middleware.go  # Bearer extraction, timeout checks
 |    |    +-- rbac/                    # Role and permission enforcement
 |    |    |    +-- rbac.go            # Permission matrix, RequireRole/RequirePermission
 |    |    |    +-- scope.go           # Object-level city/dept scope filtering
 |    |    +-- models/                  # GORM model definitions (27 tables)
 |    |    |    +-- user.go            # User with failed_attempts, locked_until
 |    |    |    +-- session.go         # JWT sessions with idle/absolute tracking
 |    |    |    +-- org.go             # OrgNode tree + ContextAssignment
 |    |    |    +-- master.go          # MasterRecord, MasterVersion, VersionItem, DeactivationEvent
 |    |    |    +-- ingestion.go       # ImportSource, IngestionJob, Checkpoint, Failure
 |    |    |    +-- media.go           # MediaAsset
 |    |    |    +-- analytics.go       # AnalyticsKPIDefinition, ReportSchedule, ReportRun
 |    |    |    +-- audit.go           # AuditLog, AuditDeleteRequest
 |    |    |    +-- integration.go     # IntegrationEndpoint, IntegrationDelivery, ConnectorDefinition
 |    |    |    +-- security.go        # SensitiveFieldRegistry, KeyRing, PasswordResetRequest
 |    |    |    +-- retention.go       # RetentionPolicy, LegalHold, PurgeRun
 |    |    +-- handlers/                # HTTP handler layer (thin, delegates to services)
 |    |    |    +-- auth_handler.go     # Login/logout/refresh
 |    |    |    +-- org_handler.go      # Org tree CRUD
 |    |    |    +-- master_handler.go   # Record CRUD + import + duplicates
 |    |    |    +-- version_handler.go  # Draft/review/activate/rollback
 |    |    |    +-- ingestion_handler.go # Sources + jobs + checkpoints
 |    |    |    +-- playback_handler.go # Media + lyrics + streaming
 |    |    |    +-- analytics_handler.go # KPIs + trends
 |    |    |    +-- report_handler.go   # Schedules + runs + download
 |    |    |    +-- audit_handler.go    # Logs + delete requests
 |    |    |    +-- security_handler.go # Sensitive fields + keys + retention + purge
 |    |    |    +-- integration_handler.go # Endpoints + deliveries + connectors
 |    |    +-- org/org_service.go       # Org tree CRUD, context switching
 |    |    +-- masterdata/              # Master record + version services
 |    |    |    +-- master_service.go   # CRUD, import, duplicate detection
 |    |    |    +-- version_service.go  # Draft/review/activate/rollback lifecycle
 |    |    +-- ingestion/               # Job engine, connectors, scheduler, validation
 |    |    |    +-- connector.go        # Connector interface + FolderConnector + ShareConnector + DatabaseConnector
 |    |    |    +-- job_engine.go       # Enqueue, execute, retry, acknowledge, dependency check
 |    |    |    +-- scheduler.go        # Cron scheduling, missed runs, retry promoter
 |    |    |    +-- validation.go       # Per-entity validation rules, CSV/XLSX parsing
 |    |    +-- playback/                # Media assets, LRC parser
 |    |    |    +-- playback_service.go # CRUD, format validation
 |    |    |    +-- lrc_parser.go       # LRC parsing (line+word level), search, UTF-16 handling
 |    |    +-- analytics/analytics_service.go  # KPI computation, trend aggregation
 |    |    +-- reports/                 # Schedule management, report generation
 |    |    |    +-- report_service.go   # Schedule CRUD, run management
 |    |    |    +-- scheduler.go        # Cron-based report execution
 |    |    +-- integration/integration_service.go  # Webhook delivery, connector CRUD, LAN validation
 |    |    +-- audit/audit_service.go   # Append-only audit logging, query/filter
 |    |    +-- security/                # Security administration
 |    |         +-- encryption.go       # AES-256-GCM encrypt/decrypt, key gen, key wrapping
 |    |         +-- security_service.go # Sensitive fields, masking, key rotation, retention, purge
 |    +-- migrations/init.sql           # Full DDL + seed data
 |    +-- tests/
 |         +-- unit/                    # Pure unit tests (no DB)
 |         +-- api/                     # Integration tests with in-memory router
 |
 +-- frontend/
 |    +-- src/
 |    |    +-- api/                     # Axios service modules per domain
 |    |    |    +-- client.js           # Base Axios instance with auth interceptors
 |    |    |    +-- auth.js, org.js, master.js, versions.js, ingestion.js,
 |    |    |       playback.js, analytics.js, reports.js, audit.js,
 |    |    |       security.js, integrations.js
 |    |    +-- components/common/       # Reusable UI primitives
 |    |    |    +-- AppButton, AppChip, AppDialog, AppInput, AppSelect,
 |    |    |       AppTable, AppToast, AppBreadcrumb, AppFileUpload,
 |    |    |       AppEmptyState, AppErrorState, AppLoadingState
 |    |    +-- pages/                   # Route-level page components
 |    |    |    +-- LoginPage, OrgTreePage, MasterDataPage, PlaybackPage,
 |    |    |       AnalyticsPage, IngestionPage, ReportsPage, SecurityAdminPage
 |    |    +-- stores/                  # Pinia stores
 |    |    |    +-- auth.js             # Token management, idle tracking, role checks
 |    |    |    +-- context.js          # Org context switching
 |    |    +-- router/index.js          # Route definitions with role-based beforeEach guard
 |    |    +-- styles/
 |    |    |    +-- variables.scss      # Design tokens (colors, spacing, typography)
 |    |    |    +-- global.scss         # Reset, base styles
 |    |    |    +-- components.scss     # Shared component styles
 |    |    +-- main.js                  # App bootstrap (createApp, Pinia, Router)
 |    +-- nginx.conf                    # Reverse proxy config + security headers
 |    +-- vite.config.js                # Dev server and build config
 |    +-- vitest.config.js              # Test runner config
 |
 +-- docker-compose.yml                 # 3-service orchestration
 +-- scripts/                           # dev-up.sh, lint.sh, test.sh
```

## Data Flow Diagram

```
                        +-------------------+
                        |   Browser (SPA)   |
                        |   Vue 3 + Pinia   |
                        +--------+----------+
                                 |
                         HTTPS / :3000
                                 |
                        +--------v----------+
                        |      Nginx        |
                        |  (reverse proxy)  |
                        |  /api/* -> :8080  |
                        | Security headers: |
                        |  X-Frame-Options  |
                        |  X-XSS-Protection |
                        |  X-Content-Type   |
                        |  Referrer-Policy  |
                        +--------+----------+
                                 |
                          HTTP / :8080
                                 |
            +--------------------v---------------------+
            |              Gin Router                   |
            |  RequestID -> Recovery -> Logger -> CORS  |
            |           -> EgressGuard                  |
            +----+--------+--------+--------+----------+
                 |        |        |        |
          +------v--+ +---v----+ +v------+ +v---------+
          |  Auth   | | RBAC   | |Errors | |Middleware|
          |Middleware| |Guards  | |Handler| |  Stack   |
          +---------+ +--------+ +-------+ +----------+
                 |
    +------------+-------------+----------------+
    |            |             |                 |
+---v---+  +----v----+  +-----v------+  +-------v--------+
| Auth  |  | Master  |  | Ingestion  |  | Security/Audit |
|Handler|  | Data    |  | Pipeline   |  | Admin          |
+---+---+  | Handler |  | Handler    |  | Handler        |
    |      +----+----+  +-----+------+  +-------+--------+
    |           |              |                 |
+---v---+  +---v-----+  +----v------+   +------v-------+
| Auth  |  | Master  |  | JobEngine |   | Security     |
|Service|  | Service |  | Scheduler |   | Service      |
+---+---+  | Version |  | Connector |   | Audit Svc    |
    |      | Service |  | Validator |   | Encryption   |
    |      +---------+  +-----------+   +--------------+
    |           |              |                 |
    +-----+----+----+---------+--------+--------+
          |         |                  |
     +----v----+  +-v-------+   +-----v------+
     |  GORM   |  | File    |   | Cron       |
     |  MySQL  |  | System  |   | Scheduler  |
     | 27 tbl  |  | (media, |   | (robfig)   |
     +---------+  | exports)|   +------------+
                  +---------+
```

## Ingestion Orchestration Flow

```
 +-----------+       +------------+       +-------------+
 | Scheduler |------>| EnqueueJob |------>| Job Queue   |
 | (cron)    |       | (validate  |       | (MySQL tbl) |
 |           |       |  source)   |       | priority    |
 +-----------+       +------------+       | DESC, FIFO  |
       |                                  +------+------+
       |  On startup                             |
       +---> HandleMissedRuns()                  |
             (catch-up-once policy)    +---------v----------+
                                       | ProcessNextJob()   |
                                       | 1. Starvation boost|
                                       |    (+10 after 30m) |
                                       | 2. Pick highest    |
                                       |    priority ready  |
                                       | 3. Check deps      |
                                       | 4. ExecuteJob()    |
                                       +---------+----------+
                                                 |
                              +------------------v------------------+
                              |           ExecuteJob()              |
                              | 1. State -> running                 |
                              | 2. Load ImportSource                |
                              | 3. Create Connector via Factory     |
                              |    (folder / network_share / database)|
                              | 4. Resume from last checkpoint      |
                              | 5. Pull batches in loop:            |
                              |    - connector.Pull(cursor, 1000)   |
                              |    - Checkpoint every 1000 records  |
                              |    - connector.AcknowledgeCheckpoint|
                              | 6. Final checkpoint                 |
                              | 7. State -> completed               |
                              +------------------+------------------+
                                                 |
                                        On failure:
                              +------------------v------------------+
                              |       handleJobFailure()            |
                              | retry_count < max_retries (3)?      |
                              |   YES -> state=retrying             |
                              |          backoff: 1m, 5m, 10m      |
                              |   NO  -> state=failed_awaiting_ack  |
                              |          (operator must acknowledge) |
                              +-------------------------------------+

 Retry Promoter (every 30s):
   Moves retrying jobs with next_retry_at <= now back to "ready"

 Job Processor (every 10s):
   Picks the highest-priority ready job and executes it
```

## Authentication Flow

```
 Client                    Backend                       Database
   |                          |                              |
   |  POST /auth/login        |                              |
   |  {username, password}    |                              |
   |------------------------->|                              |
   |                          |  SELECT * FROM users         |
   |                          |  WHERE username = ?          |
   |                          |----------------------------->|
   |                          |                              |
   |                          |  Check IsAccountLocked()     |
   |                          |  VerifyPassword(argon2id)    |
   |                          |                              |
   |                          |  GenerateTokenPair()         |
   |                          |  (access: 30m, refresh: 12h) |
   |                          |                              |
   |                          |  INSERT INTO sessions        |
   |                          |  (jwt_jti, issued_at, ...)   |
   |                          |----------------------------->|
   |                          |                              |
   |                          |  INSERT INTO audit_logs      |
   |                          |  (LOGIN action)              |
   |                          |----------------------------->|
   |                          |                              |
   |  {token, refreshToken,   |                              |
   |   user}                  |                              |
   |<-------------------------|                              |
   |                          |                              |
   |  GET /api/v1/master/sku  |                              |
   |  Authorization: Bearer   |                              |
   |------------------------->|                              |
   |                          |  ValidateToken(jwt HMAC-256) |
   |                          |  IsSessionRevoked(jti)?      |
   |                          |  CheckIdleTimeout(30m)?      |
   |                          |  CheckAbsoluteTimeout(12h)?  |
   |                          |  Update last_activity_at     |
   |                          |  Load user, check locked     |
   |                          |  Set context (user, claims)  |
   |                          |  -> RequireRole / Permission |
   |                          |  -> Handler -> Service       |
```

## Version Lifecycle

```
  draft  -->  review  -->  active  -->  archived
    ^                         |             |
    |                         |             |
    +--- (create new) --------+  rollback --+
```

State transitions enforced in `backend/internal/masterdata/version_service.go`:
- `CreateDraft()`: creates version in `draft` state
- `SubmitForReview()`: `draft` -> `review` (reviewer must differ from creator)
- `Activate()`: `review` -> `active` (SystemAdmin only, archives current active version atomically)
- `Rollback()`: reactivates an `archived` version, archives current `active`

## Deployment Topology (Docker Compose)

| Service | Container | Port | Image | Health Check |
|---|---|---|---|---|
| MySQL | hub-mysql | 3306:3306 | mysql:8.0 | `mysqladmin ping` every 10s |
| Backend | hub-backend | 8080:8080 | Custom Go build | Depends on MySQL healthy |
| Frontend | hub-frontend | 3000:80 | Custom Node+Nginx | Depends on Backend |

## Key Design Decisions

1. **Service layer pattern**: Handlers are thin HTTP adapters; business logic lives in service packages (`masterdata`, `ingestion`, `security`, etc.).

2. **GORM auto-migration + SQL init**: The initial schema is applied via Docker entrypoint (`init.sql`), while GORM `AutoMigrate` handles model sync on startup.

3. **Connector plugin interface**: `backend/internal/ingestion/connector.go` defines the `Connector` interface with `Type()`, `Capabilities()`, `ValidateConfig()`, `HealthCheck()`, `Pull()`, and `AcknowledgeCheckpoint()`. New connectors implement this interface.

4. **Append-only audit**: `audit_service.go` only calls `db.Create()`. No `UPDATE` or `DELETE` on audit_logs. Deletion requires a dual-approval workflow through `audit_delete_requests`. The security purge endpoint explicitly rejects `audit_logs` as a target — all audit log deletion must go through dual-approval.

8. **Scheduler startup**: Both the ingestion scheduler and report scheduler are instantiated and started in `main.go` after DB initialization but before the HTTP server starts. On shutdown (SIGINT/SIGTERM), both schedulers are stopped gracefully before the HTTP server.

9. **CORS configuration**: CORS origins are configured via the `CORS_ORIGINS` environment variable (comma-separated). Wildcard origins are not used with credentials. Default: `http://localhost:3000`.

10. **Scope enforcement**: Analytics and report endpoints enforce object-level scope from the authenticated user's DB record (city_scope, dept_scope), never from query parameters. Non-admin users with no scope assigned are denied by default.

5. **Scope-aware queries**: `rbac/scope.go` provides `EnforceScopeOnQuery()` which appends WHERE clauses for city/department, applied at the service layer.

6. **LAN-only egress**: `middleware/middleware.go` EgressGuard checks client IPs against RFC 1918 ranges. `integration/integration_service.go` ValidateEndpointURL blocks non-private webhook targets.

7. **Structured error contract**: All errors use `AppError{code, message, details, correlationId}` via `errors/errors.go`.
