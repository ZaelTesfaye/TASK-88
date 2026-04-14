# Fix Check Report for audit_report-3-fix_check.md (Static Re-Inspection)

Date: 2026-04-14
Mode: Static only (no runtime, no tests, no Docker)
Source baseline reviewed: .tmp/audit_report-3-fix_check.md

## Overall Result

- Fixed: 9
- Partially Fixed: 0
- Not Fixed: 0

## Issue-by-Issue Re-Validation

### 1) High - Report object-level authorization missing on list/get endpoints

- Previous status: Partially Fixed
- Current status: Fixed
- Why:
  - Handler still applies scope to list and object access on get.
  - New integration-style list test now asserts cross-scope filtering behavior for GET /reports/runs.
- Evidence:
  - [repo/backend/internal/handlers/report_handler.go](repo/backend/internal/handlers/report_handler.go#L209)
  - [repo/backend/internal/handlers/report_handler.go](repo/backend/internal/handlers/report_handler.go#L230)
  - [repo/backend/internal/handlers/report_handler.go](repo/backend/internal/handlers/report_handler.go#L231)
  - [repo/backend/internal/handlers/report_handler.go](repo/backend/internal/handlers/report_handler.go#L243)
  - [repo/backend/tests/api/scope_enforcement_test.go](repo/backend/tests/api/scope_enforcement_test.go#L140)
  - [repo/backend/tests/api/scope_enforcement_test.go](repo/backend/tests/api/scope_enforcement_test.go#L168)

### 2) High - Master-data scope can degrade to broad access when context assignment is missing

- Previous status: Fixed
- Current status: Fixed
- Evidence:
  - [repo/backend/internal/handlers/master_handler.go](repo/backend/internal/handlers/master_handler.go#L389)
  - [repo/backend/internal/handlers/master_handler.go](repo/backend/internal/handlers/master_handler.go#L404)
  - [repo/backend/internal/masterdata/master_service.go](repo/backend/internal/masterdata/master_service.go#L174)
  - [repo/backend/internal/masterdata/master_service.go](repo/backend/internal/masterdata/master_service.go#L175)

### 3) High - Playback write operations allowed for any authenticated role

- Previous status: Fixed
- Current status: Fixed
- Evidence:
  - [repo/backend/internal/router/router.go](repo/backend/internal/router/router.go#L170)
  - [repo/backend/internal/router/router.go](repo/backend/internal/router/router.go#L171)
  - [repo/backend/internal/router/router.go](repo/backend/internal/router/router.go#L172)
  - [repo/backend/tests/api/scope_enforcement_test.go](repo/backend/tests/api/scope_enforcement_test.go#L270)

### 4) High - Frontend/backend mismatch for lyrics parse response contract

- Previous status: Fixed
- Current status: Fixed
- Evidence:
  - [repo/backend/internal/handlers/playback_handler.go](repo/backend/internal/handlers/playback_handler.go#L270)
  - [repo/frontend/src/pages/PlaybackPage.vue](repo/frontend/src/pages/PlaybackPage.vue#L368)
  - [repo/backend/tests/api/contract_test.go](repo/backend/tests/api/contract_test.go#L260)

### 5) High - Reports frontend adapter/query/response mismatch

- Previous status: Fixed
- Current status: Fixed
- Evidence:
  - [repo/frontend/src/pages/ReportsPage.vue](repo/frontend/src/pages/ReportsPage.vue#L368)
  - [repo/frontend/src/pages/ReportsPage.vue](repo/frontend/src/pages/ReportsPage.vue#L369)
  - [repo/frontend/src/pages/ReportsPage.vue](repo/frontend/src/pages/ReportsPage.vue#L374)
  - [repo/frontend/src/pages/ReportsPage.vue](repo/frontend/src/pages/ReportsPage.vue#L399)
  - [repo/frontend/src/api/reports.js](repo/frontend/src/api/reports.js#L28)
  - [repo/backend/tests/api/contract_test.go](repo/backend/tests/api/contract_test.go#L297)
  - [repo/backend/tests/api/contract_test.go](repo/backend/tests/api/contract_test.go#L325)

### 6) Medium - Frontend reports route roles conflicted with backend authorization

- Previous status: Fixed
- Current status: Fixed
- Evidence:
  - [repo/frontend/src/router/index.js](repo/frontend/src/router/index.js#L42)
  - [repo/frontend/src/router/index.js](repo/frontend/src/router/index.js#L45)
  - [repo/backend/internal/router/router.go](repo/backend/internal/router/router.go#L194)

### 7) Medium - Scope enforcement report test was non-assertive

- Previous status: Fixed
- Current status: Fixed
- Evidence:
  - [repo/backend/tests/api/scope_enforcement_test.go](repo/backend/tests/api/scope_enforcement_test.go#L140)
  - [repo/backend/tests/api/scope_enforcement_test.go](repo/backend/tests/api/scope_enforcement_test.go#L168)

### 8) Medium - Security regression tests were simulation-heavy

- Previous status: Not Fixed
- Current status: Fixed
- Why:
  - security_regression_test.go now uses real production router + DB-backed setup throughout and explicitly documents no simulated routes.
- Evidence:
  - [repo/backend/tests/api/security_regression_test.go](repo/backend/tests/api/security_regression_test.go#L12)
  - [repo/backend/tests/api/security_regression_test.go](repo/backend/tests/api/security_regression_test.go#L14)
  - [repo/backend/tests/api/security_regression_test.go](repo/backend/tests/api/security_regression_test.go#L31)
  - [repo/backend/tests/api/security_regression_test.go](repo/backend/tests/api/security_regression_test.go#L57)

### 9) Low - Frontend compile check in run_tests.sh could not fail

- Previous status: Fixed
- Current status: Fixed
- Evidence:
  - [repo/run_tests.sh](repo/run_tests.sh#L39)
  - [repo/run_tests.sh](repo/run_tests.sh#L43)

## Conclusion

All previously listed gaps from .tmp/audit_report-3-fix_check.md are now statically verified as fixed in current code/test artifacts.
