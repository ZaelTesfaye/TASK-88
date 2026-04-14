# audit_report-2 Fix Check

Date: 2026-04-14
Source reviewed: `.tmp/audit_report-2.md`
Verification mode: static code/document inspection only (no runtime execution, no Docker startup, no DB-backed test run in this pass)

## Overall Result

- **Status: Mostly fixed (7 fixed, 1 partially fixed)**
- The previously reported high-impact implementation gaps around DB connector functionality, object/data scope enforcement, TLS wiring, key wrapping, schema drift, and security-doc consistency are now addressed in code.
- Remaining gap is test realism/completeness: real integration coverage exists now, but the suite still contains many synthetic tests and at least one integration test has weak assertions.

## Finding-by-Finding Check

### F-01 On-prem database ingestion connector is non-functional stub

- **Previous**: Fail
- **Current**: **Fixed**
- **Why**:
  - Real SQL connectivity/validation now implemented via `database/sql` and `sql.Open` in connector config validation and health checks.
  - `Pull` now performs real cursor-based query execution with incremental and backfill behavior.
- **Evidence**:
  - `repo/backend/internal/ingestion/connector.go:4`
  - `repo/backend/internal/ingestion/connector.go:489`
  - `repo/backend/internal/ingestion/connector.go:530`
  - `repo/backend/internal/ingestion/connector.go:569`
  - `repo/docs/mock-and-stub-disclosure.md:22`
  - `repo/docs/mock-and-stub-disclosure.md:44`

### F-02 Scope isolation is not enforced in key data paths

- **Previous**: Fail
- **Current**: **Fixed**
- **Why**:
  - Master record detail/update/deactivate paths now perform object-level scope checks via `isRecordInScope`.
  - Analytics reads scoped values from auth context (`city_scope`, `dept_scope`) and applies filtering in service query logic.
  - Report generation applies `scope_json` city/department filters to dataset query; download-time access is revalidated against user scope.
- **Evidence**:
  - `repo/backend/internal/handlers/master_handler.go:166`
  - `repo/backend/internal/handlers/master_handler.go:198`
  - `repo/backend/internal/handlers/master_handler.go:248`
  - `repo/backend/internal/handlers/master_handler.go:391`
  - `repo/backend/internal/handlers/analytics_handler.go:38`
  - `repo/backend/internal/analytics/analytics_service.go:218`
  - `repo/backend/internal/analytics/analytics_service.go:230`
  - `repo/backend/internal/analytics/analytics_service.go:346`
  - `repo/backend/internal/reports/report_service.go:464`
  - `repo/backend/internal/reports/report_service.go:479`
  - `repo/backend/internal/reports/report_service.go:534`

### F-03 Critical API tests are synthetic and do not verify real handler+DB behavior

- **Previous**: Fail
- **Current**: **Partially fixed**
- **Why**:
  - Real DB + production router integration harness now exists (`getTestDB`, `realRouter`, `database.AutoMigrate`, `router.SetupRouter`).
  - Real integration tests were added for scope enforcement and DB connector behavior.
  - However, many synthetic tests still exist and some integration assertions remain weak (example: report scope test logs response but does not strictly fail on incorrect authorization behavior).
- **Evidence**:
  - `repo/backend/tests/api/integration_helpers_test.go:30`
  - `repo/backend/tests/api/integration_helpers_test.go:49`
  - `repo/backend/tests/api/integration_helpers_test.go:60`
  - `repo/backend/tests/api/scope_enforcement_test.go:20`
  - `repo/backend/tests/api/scope_enforcement_test.go:46`
  - `repo/backend/tests/api/db_connector_test.go:62`
  - `repo/backend/tests/api/helpers_test.go:24`
  - `repo/backend/tests/api/master_api_test.go:18`
  - `repo/backend/tests/api/scope_enforcement_test.go:152`

### F-04 Migration schema and model/service schema diverge materially

- **Previous**: Fail
- **Current**: **Fixed**
- **Why**:
  - `init.sql` now documents alignment intent and appears updated to current model contracts for key previously cited tables (`sessions`, `master_*`, `key_rings`, etc.).
- **Evidence**:
  - `repo/backend/migrations/init.sql:1`
  - `repo/backend/migrations/init.sql:40`
  - `repo/backend/migrations/init.sql:110`
  - `repo/backend/migrations/init.sql:432`
  - `repo/backend/internal/models/session.go:7`
  - `repo/backend/internal/models/master.go:7`
  - `repo/backend/internal/models/security.go:20`

### F-05 Frontend role matrix conflicts with backend permissions for standard user master-data viewing

- **Previous**: Partial Fail
- **Current**: **Fixed**
- **Why**:
  - Frontend route for master-data now includes `standard_user`.
  - Backend RBAC matrix includes `master_data_view` for `standard_user`.
- **Evidence**:
  - `repo/frontend/src/router/index.js:21`
  - `repo/backend/internal/rbac/rbac.go:60`

### F-06 TLS configuration flag is documented but not wired in server startup

- **Previous**: Partial Fail
- **Current**: **Fixed**
- **Why**:
  - Startup now conditionally runs `ListenAndServeTLS` when TLS is enabled and cert/key paths are present; startup fails fast if TLS env vars are incomplete.
- **Evidence**:
  - `repo/backend/cmd/server/main.go:80`
  - `repo/backend/cmd/server/main.go:84`
  - `repo/backend/cmd/server/main.go:89`

### F-07 Key rotation stores raw generated key bytes as wrapped material

- **Previous**: Partial Fail
- **Current**: **Fixed**
- **Why**:
  - Rotation now requires `MASTER_KEY_HEX`, envelope-encrypts generated data keys (`WrapKey`), and stores base64 ciphertext in `WrappedKey`.
- **Evidence**:
  - `repo/backend/internal/security/security_service.go:320`
  - `repo/backend/internal/security/security_service.go:335`
  - `repo/backend/internal/security/security_service.go:363`
  - `repo/backend/internal/security/security_service.go:373`
  - `repo/backend/internal/security/security_service.go:390`

### F-08 Security documentation claims BCrypt while implementation uses Argon2id

- **Previous**: Documentation mismatch
- **Current**: **Fixed**
- **Why**:
  - README now states Argon2id, matching implementation.
- **Evidence**:
  - `repo/README.md:225`
  - `repo/backend/internal/auth/auth_service.go:57`

## Updated Severity Outlook

- **Open High**: None from the original F-01..F-08 set.
- **Open Medium**: Test robustness gap (F-03 partial).
- **Residual risk**:
  - Integration tests depending on `TEST_DB_DSN` may be skipped in many environments.
  - Some scope-related tests remain simulated; not all critical flows are proven through strict real-stack assertions.

## Suggested Follow-up (for full closure of F-03)

1. Convert `TestReportScopeFiltering` into strict pass/fail assertions on denial/success behavior.
2. Add real-router, real-DB assertions for analytics KPI/report dataset scope correctness (not only context echoing).
3. Ensure CI provides `TEST_DB_DSN` (or testcontainers equivalent) so integration tests are always executed, not routinely skipped.
