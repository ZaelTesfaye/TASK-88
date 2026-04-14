# System Design & Architecture

## Overview

The Multi-Org Data & Media Operations Hub is a full-stack platform for managing multi-organizational data operations, master data, media assets, integrations, reporting, and compliance. The system is built with a modern three-tier architecture utilizing Vue 3 (frontend), Go with Gin framework (backend), and MySQL 8 (persistence).

### High-Level Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Frontend Layer                              │
│                      Vue 3 SPA + Nginx Proxy                         │
│                   (Port 3000, proxies /api to backend)               │
└─────────────────────────────────────────┬───────────────────────────┘
                                          │ HTTP/HTTPS
                                          ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Backend Layer (Go/Gin)                       │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  API Routes, Authentication, RBAC, Middleware               │   │
│  │  - Auth & Session Management                                │   │
│  │  - Request Validation & Error Handling                      │   │
│  │  - Audit Logging & Access Control                           │   │
│  └──────────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Business Logic Layer (Services)                             │   │
│  │  - Master Data Management               - Org Management    │   │
│  │  - Versioning & Version Control         - Playback & Media  │   │
│  │  - Ingestion Pipelines                  - Analytics & KPIs  │   │
│  │  - Report Generation & Scheduling       - Security & Audit  │   │
│  │  - Integrations & Connectors            - Retention Policy  │   │
│  └──────────────────────────────────────────────────────────────┘   │
│           (Port 8080)                                                 │
└─────────────────────────────────────────┬───────────────────────────┘
                                          │ TCP Connection
                                          ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Persistence Layer                               │
│                   MySQL 8.0 (Port 3306)                              │
│  - User & Authentication data                                        │
│  - Organization hierarchy & context assignments                      │
│  - Master records, versions, & history                               │
│  - Media assets & metadata                                           │
│  - Ingestion jobs & connectors                                       │
│  - Analytics & KPI data                                              │
│  - Audit logs & compliance records                                   │
│  - Report schedules & executions                                     │
│  - Security keys, sensitive field encryption metadata               │
└─────────────────────────────────────────────────────────────────────┘
```

## Layered Architecture

### Frontend Layer (Vue 3 + Vite + Nginx)

**Purpose:** Deliver an interactive single-page application for organizing, managing, and viewing data.

**Key Responsibilities:**
- User authentication (login/logout, token refresh)
- Client-side role-based route guards
- Form validation and submission
- Data visualization (reports, analytics, media playback)
- Session state management
- API client abstraction with request interceptors

**Technology Stack:**
- Vue 3 with Composition API
- Vite for fast development and optimized bundling
- Nginx reverse proxy for serving static assets and proxying API requests to backend
- Axios for HTTP client library

**Directory Structure:**
```
frontend/
├── src/
│   ├── api/               # API adapter layer (exports functions for backend calls)
│   │   ├── client.js      # Axios instance with JWT interceptor
│   │   ├── auth.js        # /auth/* endpoints
│   │   ├── master.js      # Master data endpoints
│   │   ├── playback.js    # Media/playback endpoints
│   │   ├── reports.js     # Report endpoints
│   │   └── ...
│   ├── components/        # Reusable Vue components
│   ├── pages/             # Page-level components (one per route)
│   │   ├── LoginPage.vue
│   │   ├── MasterDataPage.vue
│   │   ├── PlaybackPage.vue
│   │   ├── ReportsPage.vue
│   │   └── ...
│   ├── router/            # Vue Router configuration with role guards
│   ├── stores/            # Pinia store modules (auth, user state)
│   ├── styles/            # Global styles & SCSS variables
│   ├── App.vue            # Root component
│   └── main.js            # Entry point
├── package.json           # Dependencies
├── vite.config.js         # Vite build configuration
└── Dockerfile             # Static build & Nginx serving image
```

### Backend Layer (Go + Gin Framework)

**Purpose:** Serve RESTful API endpoints, enforce authentication, authorization, data validation, and business logic.

**Key Responsibilities:**
- Request routing and handler dispatch
- JWT authentication and session management
- RBAC (Role-Based Access Control) enforcement
- Data persistence via GORM ORM
- Service layer for business logic
- Error handling and consistent error response formatting
- Audit logging of sensitive actions
- Scheduled jobs (report generation, ingestion processing)

**Technology Stack:**
- Go 1.25+ for high concurrency and minimal memory footprint
- Gin web framework for HTTP routing and middleware composition
- GORM for type-safe database abstraction
- JWT (golang-jwt/jwt) for stateless token signing/validation
- go.uber.org/zap for structured JSON logging
- robfig/cron for job scheduling (cron expressions)

**Directory Structure:**
```
backend/
├── cmd/
│   └── server/            # Application entry point
│       └── main.go        # Server initialization, scheduler setup
├── internal/
│   ├── auth/              # JWT validation, session management
│   │   ├── auth_middleware.go
│   │   ├── auth_service.go
│   │   └── claims.go
│   ├── handlers/          # HTTP request handlers
│   │   ├── master_handler.go
│   │   ├── report_handler.go
│   │   ├── playback_handler.go
│   │   ├── analytics_handler.go
│   │   ├── audit_handler.go
│   │   ├── security_handler.go
│   │   └── ...
│   ├── services/          # Business logic, DB queries
│   │   ├── master_service.go
│   │   ├── reports/
│   │   ├── analytics/
│   │   ├── ingestion/
│   │   ├── security/
│   │   └── ...
│   ├── models/            # GORM model definitions
│   │   ├── user.go
│   │   ├── master.go
│   │   ├── report.go
│   │   ├── media.go
│   │   └── ...
│   ├── rbac/              # Role-based access control middleware
│   │   ├── rbac.go        # Role definitions & permission matrix
│   │   └── scope.go       # Scope enforcement for multi-org data
│   ├── middleware/        # Gin middleware (request ID, recovery, logging)
│   ├── router/            # Route definitions
│   │   └── router.go      # registerAllRoutes() with route groups
│   ├── database/          # Database connection & auto-migration
│   │   └── database.go
│   ├── config/            # Environment variable parsing
│   │   └── config.go
│   ├── logging/           # Structured logging with zap
│   │   └── logger.go
│   ├── errors/            # Error types and response helpers
│   │   └── errors.go
│   └── migrations/        # SQL migration scripts
│       └── init.sql
├── tests/                 # Integration and unit tests
│   ├── api/               # API integration tests
│   │   ├── scope_enforcement_test.go
│   │   ├── security_regression_test.go
│   │   └── contract_test.go
│   └── unit/              # Unit tests for services
├── go.mod                 # Go module definition
└── go.sum                 # Dependency lock file
```

### Persistence Layer (MySQL 8.0)

**Purpose:** Reliable, ACID-compliant storage of all application data.

**Key Responsibilities:**
- Persistent storage of all domain entities
- Referential integrity via foreign keys
- Efficient querying via indexes
- Transaction support for multi-step operations
- Automated backups and point-in-time recovery

**Database Schema (27 Tables):**

**Authentication & Users:**
- `users` — User accounts with credentials, roles, and scopes
- `sessions` — Active JWT sessions with timeout tracking

**Organization:**
- `org_nodes` — Hierarchical organizational structure
- `context_assignments` — Mapping of users to org nodes (scope)

**Master Data Management:**
- `master_records` — Entity records (SKU, location, customer, etc.)
- `master_versions` — Versioned snapshots of records
- `master_version_items` — Items included in a version
- `deactivation_events` — History of record deactivations

**Media & Playback:**
- `media_assets` — Audio/video files with metadata
- `lyrics` — LRC format lyrics with timestamps

**Ingestion & Integration:**
- `import_sources` — Data source configurations
- `ingestion_jobs` — Data ingestion job executions
- `ingestion_checkpoints` — Progress tracking for long-running jobs
- `ingestion_failures` — Error records for troubleshooting
- `connectors` — Third-party integration endpoints
- `integration_deliveries` — Delivery attempts to external systems

**Analytics & Reporting:**
- `analytics_kpi_definitions` — KPI metric definitions
- `analytics_kpi_values` — Computed KPI scores over time
- `report_schedules` — CRON-based report scheduling
- `report_runs` — Generated report instances
- `retention_policies` — Data retention rules by artifact type
- `legal_holds` — Blocks on data deletion for compliance

**Compliance & Security:**
- `audit_logs` — Immutable action records (who, what, when, where)
- `delete_requests` — Tracked deletion approvals (dual-control)
- `sensitive_fields` — Field encryption metadata
- `encryption_keys` — Master keys for field-level encryption

## Key Design Decisions

### 1. Token-Based Stateless Authentication (JWT)

**Decision:** Use JWT (JSON Web Tokens) with a single session table for revocation/timeout tracking.

**Rationale:**
- Stateless design allows horizontal scaling without sticky sessions
- JTI (JWT ID) claim enables revocation by flagging session records
- Claims embed role and scope to avoid repeated DB lookups
- Supports both short-lived access tokens and refresh token patterns
- No session affinity required in load-balanced deployments

**Implementation:**
- Tokens signed with HMAC-SHA256 using `JWT_SECRET`
- Session timeout: 30 minutes of inactivity (idle timeout)
- Absolute timeout: 24 hours from token creation
- Claims include: `sub` (user_id), `role`, `city_scope`, `dept_scope`, `jti` (session_id)

### 2. Role-Based Access Control (RBAC)

**Decision:** Implement explicit role hierarchy with permission matrix rather than per-resource access lists.

**Rationale:**
- Simpler governance: permissions map to easily understood business roles
- Low database overhead: no per-resource ACL overhead
- Clear audit trail: actions explicitly tied to roles and permissions
- Scales better: adding resources doesn't linearly increase ACL entries

**Roles:**
- `system_admin` — Full platform access, org management, security config
- `data_steward` — Master data management, versioning, data imports
- `operations_analyst` — Reporting, analytics, ingestion oversight
- `standard_user` — Reporting view, media playback, master data read-only

**Permission Matrix:**
```
Permission              | system_admin | data_steward | ops_analyst | standard_user
─────────────────────────────────────────────────────────────────────────────────
master_data_view        | ✓            | ✓            | ✓           | ✓
master_data_crud        | ✓            | ✓            | ✗           | ✗
master_data_import      | ✓            | ✓            | ✗           | ✗
version_draft           | ✓            | ✓            | ✗           | ✗
version_activate        | ✓            | ✗            | ✗           | ✗
analytics_view          | ✓            | ✗            | ✓           | ✗
reports_view            | ✓            | ✗            | ✓           | ✓
reports_download        | ✓            | ✗            | ✓           | ✗
ingestion_view          | ✓            | ✗            | ✓           | ✗
ingestion_manage        | ✓            | ✗            | ✗           | ✗
audit_view              | ✓            | ✗            | ✗           | ✗
audit_manage            | ✓            | ✗            | ✗           | ✗
security_manage         | ✓            | ✗            | ✗           | ✗
org_manage              | ✓            | ✗            | ✗           | ✗
integration_manage      | ✓            | ✗            | ✗           | ✗
```

### 3. Multi-Organizational Scope Enforcement

**Decision:** Implement scope as orthogonal to RBAC using context assignments linking users to org nodes.

**Rationale:**
- Allows same role (e.g., analyst) in different orgs to access only their org's data
- Scope enforced at query level: filters joined data, no post-fetch filtering
- Fail-closed design: missing context assignment denies access
- Supports hierarchical scoping (city → department)

**Implementation:**
- Users have `city_scope` and `department_scope` string fields
- `context_assignments` table maps users to `org_nodes` (fine-grained scope)
- Handlers extract scope keys from authenticated user before querying
- Sensitive endpoints (reports, analytics) enforce scope via `EnforceScopeContext()` middleware
- Cross-scope access returns 403 Forbidden

### 4. Versioning & Dual-Control Approval

**Decision:** Master records are immutable; changes tracked via versions with approval workflow.

**Rationale:**
- Preserves complete audit trail of changes
- Enables rollback to previous versions
- Dual-approval required for activation prevents accidental rollout
- Supports parallel drafts from different users
- Decouples drafting from approval for governance

**Workflow:**
1. User creates new `master_version` in draft state
2. User adds/removes `master_version_items` and submits for review
3. Reviewer examines diff and approves or rejects
4. On approval, version transitions to active state
5. Previous active version automatically archived

### 5. Scheduled Jobs with Cron Expressions

**Decision:** Use cron expression scheduling with a persistent scheduler (not just external cron).

**Rationale:**
- Jobs survive server restarts (state in DB)
- Configurable per-schedule timezone support
- Consistent job execution guarantees
- Audit-friendly: all runs logged with start/end times
- Supports catch-up for missed runs during downtime

**Implementations:**
- **Report Scheduler:** Generates reports on schedule, stores as downloadable files
- **Ingestion Scheduler:** Pulls data from sources, transforms, validates, persists
- Both recover from outages using `HandleMissedRuns()` per configured policy

### 6. Field-Level Encryption for Sensitive Data

**Decision:** Encrypt sensitive fields at rest using master key rotation strategy.

**Rationale:**
- Granular control over which fields are encrypted (vs. whole-row TDE)
- Key rotation doesn't require re-encryption of all historical data
- Encrypted at application layer: DB sees only ciphertext
- Compliance-friendly: separate keys for different sensitivity levels

**Implementation:**
- Master key stored securely (env var or external vault in production)
- Each encryption record tracks which key version encrypted it
- Rotation scheduled per `KEY_ROTATION_DAYS` config
- Old key versions retained for decryption of historical data

### 7. Dual-Control Delete Requests

**Decision:** Require two independent approvals for audit log deletions and data purges.

**Rationale:**
- Prevents accidental loss of compliance-critical records
- Distributes power: no single admin can unilaterally delete audit logs
- Clear audit trail of who requested and approved deletion
- Configurable retention policies per data type

**Implementation:**
- User creates `delete_request` with reason and data scope
- Admin 1 reviews and approves
- Admin 2 confirms the approval before execution
- Execute deletes only if both approvals in place
- Audit log entry recorded with both approvers

## Technology Stack Justification

### Go Language
- **Why:** High concurrency, minimal memory footprint, single binary deployment
- **Performance:** Faster than Node.js/Python for I/O-bound operations (database queries)
- **Ops:** No runtime dependencies; cross-platform compilation
- **Scalability:** Goroutines enable thousands of concurrent connections per server

### Gin Web Framework
- **Why:** Lightweight, high-performance, minimal boilerplate
- **Features:** Built-in middleware support, context management, JSON binding
- **Alternative Considered:** Echo (similar, chose Gin for larger ecosystem)

### MySQL 8.0
- **Why:** ACID compliance for financial/compliance data, relational model matches domain
- **Features:** JSON column support for flexible metadata, GROUP_CONCAT for hierarchies
- **Alternative Considered:** PostgreSQL (excellent but MySQL more common in enterprises)

### Vue 3 + Vite
- **Why:** Modern, reactive framework; Vite provides instant HMR during development
- **Performance:** Tree-shaking, code-splitting, lazy route loading
- **Developer Experience:** TypeScript support, composition API simpler than React hooks

### Nginx Reverse Proxy
- **Why:** Lightweight, high-throughput HTTP server; perfect for static asset serving + API proxying
- **Features:** Built-in compression, caching, SSL termination, load balancing

## Data Flow: Request Lifecycle

```
1. Client Request (Frontend)
   ├─ GET /api/v1/reports/runs
   ├─ Headers: Authorization: Bearer <jwt_token>
   └─ Body: (none for GET)

2. Nginx (Frontend Container)
   ├─ Receive request on port 3000
   ├─ Match route against /api/* proxy rule
   └─ Forward to backend:8080

3. Backend Ingress
   ├─ Request Logger Middleware (logs all requests)
   ├─ Recovery Middleware (catches panics)
   ├─ Request ID Middleware (generates correlation ID)
   ├─ CORS Middleware (enforces origin allowlist)
   └─ Egress Guard Middleware (validates client IP against ALLOWED_HOSTS)

4. Route Dispatch (router.go)
   ├─ Route: GET /api/v1/reports/runs
   ├─ Group: /reports (role-protected: system_admin OR operations_analyst)
   ├─ Middleware: RBACEnforceScopeContext() 
   └─ Handler: reportHandler.ListRuns()

5. Auth Middleware (auth.AuthRequired)
   ├─ Extract Bearer token from Authorization header
   ├─ Validate JWT signature using JWT_SECRET
   ├─ Check claims: role, city_scope, dept_scope
   ├─ Query db.sessions: verify session not revoked
   ├─ Check idle timeout (30 min) and absolute timeout (24h)
   ├─ Update session.last_activity_at
   ├─ Load user record from db.users
   ├─ Check account not locked
   └─ Store in context: current_user, current_claims, user_role, city_scope, dept_scope

6. RBAC Role Middleware (rbac.RequireRole)
   ├─ Extract user_role from context
   ├─ Verify role ∈ [system_admin, operations_analyst]
   └─ If missing role → 403 Forbidden

7. Scope Enforcement Middleware (rbac.EnforceScopeContext)
   ├─ Check if user is system_admin (bypass scope checks)
   ├─ Otherwise, verify city_scope and dept_scope are set
   ├─ If no scope → 403 Forbidden
   └─ Store scope in context for handler use

8. Handler (reportHandler.ListRuns)
   ├─ Parse query params: schedule_id, date_from, date_to, page, page_size
   ├─ Extract scope from context: city_scope, dept_scope
   ├─ Inject scope into RunFilter: UserCity, UserDept
   ├─ Call service.GetRuns(filter)
   └─ Return: {"items": [...], "total": count}

9. Service Layer (reports.ReportService.GetRuns)
   ├─ Build GORM query on db.report_runs
   ├─ WHERE schedule_id = ? (if provided)
   ├─ WHERE state = ? (if provided)
   ├─ WHERE created_at >= ? (date_from)
   ├─ WHERE created_at <= ? (date_to)
   ├─ WHERE schedule_id IN (
   │    SELECT id FROM report_schedules 
   │    WHERE scope_json->'$.city' = UserCity OR scope_json IS NULL
   │ ) — Scope filter
   ├─ Apply pagination (LIMIT, OFFSET)
   ├─ Execute query
   └─ Marshal to []models.ReportRun

10. Response (Handler Marshal → Gin)
    ├─ HTTP Status: 200 OK
    ├─ Content-Type: application/json
    ├─ Body: {"items": [...runs...], "total": 42}
    └─ Headers: X-Correlation-ID (matches request ID)

11. Nginx (Response)
    ├─ Receive response from backend
    ├─ Add CORS headers (Access-Control-Allow-Origin)
    ├─ Compress response (gzip)
    └─ Send to client on port 3000

12. Frontend (Success Handler)
    ├─ Receive 200 OK with data
    ├─ Update reactive state with runs
    ├─ Render table/list view
    └─ Display to user
```

## Error Handling & Response Format

All API errors follow a consistent structure:

```json
{
  "code": "FORBIDDEN",
  "message": "insufficient role privileges",
  "details": null,
  "correlationId": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Error Code Reference:**
- `AUTH_REQUIRED` (401) — Missing or invalid JWT token
- `FORBIDDEN` (403) — Valid token but insufficient role/permission
- `BAD_REQUEST` (400) — Malformed request body or invalid parameters
- `NOT_FOUND` (404) — Resource does not exist
- `CONFLICT` (409) — Resource already exists or state conflict
- `VALIDATION_ERROR` (422) — Business logic validation failed
- `INTERNAL_ERROR` (500) — Unexpected server error

## Deployment & Scalability

### Scaling Considerations

1. **Horizontal Scaling (Multiple Backends)**
   - JWT stateless design enables any backend instance to serve any request
   - Session table is shared source of truth for revocation
   - No sticky sessions required

2. **Database Connection Pooling**
   - GORM configured with 25 max open connections, 10 idle
   - Each backend instance gets its own pool
   - Total DB connections = (num backends) × 25

3. **Caching Opportunities**
   - Org hierarchy changes infrequently → cache `org_nodes` tree
   - Permission matrix static → cache in memory per process
   - Analytics KPIs computed periodically → cache results with TTL

4. **Async Job Processing**
   - Report generation moved to background scheduler (doesn't block HTTP request)
   - Ingestion jobs run on configurable intervals with persistent checkpoints
   - Supports retry logic for transient failures

### Production Deployment

**Environment Variables (Critical):**
- `JWT_SECRET` — Use strong random value (min 32 chars, rotate periodically)
- `DB_PASSWORD` — Store in secrets manager, never commit to code
- `MASTER_KEY_HEX` — Encryption key for sensitive fields (external vault recommended)
- `ALLOWED_HOSTS` — Restrict to known IP ranges (VPN, corporate networks)
- `CORS_ORIGINS` — Use exact domain (never wildcard with credentials)

**Monitoring & Observability:**
- Structured JSON logging shipped to log aggregator (Splunk, ELK, etc.)
- Correlation IDs in all logs enable request tracing
- Audit logs queryable for compliance reporting
- Metrics exported for alerting (request latency, error rates, DB pool stats)
