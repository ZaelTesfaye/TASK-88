# Fix Check Report for audit_report-3.md (Static Re-Inspection)

Date: 2026-04-14
Mode: Static only (no runtime, no tests, no Docker)
Source baseline reviewed: .tmp/audit_report-3.md

## Overall Result

- Fixed: 7
- Partially Fixed: 1
- Not Fixed: 1

## Issue-by-Issue Re-Validation

### 1) High - Report object-level authorization missing on list/get endpoints

- Previous finding: List and Get report-run endpoints did not enforce scope/object access.
- Current status: Partially Fixed
- What changed:
  - GetRun now calls CheckAccess before returning run data.
  - ListRuns now injects user scope into filter and service applies scope filtering.
- Evidence:
  - [repo/backend/internal/handlers/report_handler.go](repo/backend/internal/handlers/report_handler.go#L243)
  - [repo/backend/internal/handlers/report_handler.go](repo/backend/internal/handlers/report_handler.go#L276)
  - [repo/backend/internal/handlers/report_handler.go](repo/backend/internal/handlers/report_handler.go#L209)
  - [repo/backend/internal/reports/report_service.go](repo/backend/internal/reports/report_service.go#L56)
  - [repo/backend/internal/reports/report_service.go](repo/backend/internal/reports/report_service.go#L210)
- Remaining gap:
  - No direct backend test found that asserts cross-scope denial/filtering behavior specifically for GET /reports/runs list results (GetRun is now asserted).

### 2) High - Master-data scope can degrade to broad access when context assignment is missing

- Previous finding: Missing user context could lead to permissive behavior.
- Current status: Fixed
- What changed:
  - isRecordInScope now fail-closes when context assignment is missing.
  - ListRecords adds fallback city/department scope filtering when NodeIDs are unavailable.
- Evidence:
  - [repo/backend/internal/handlers/master_handler.go](repo/backend/internal/handlers/master_handler.go#L404)
  - [repo/backend/internal/handlers/master_handler.go](repo/backend/internal/handlers/master_handler.go#L406)
  - [repo/backend/internal/masterdata/master_service.go](repo/backend/internal/masterdata/master_service.go#L174)
  - [repo/backend/internal/masterdata/master_service.go](repo/backend/internal/masterdata/master_service.go#L180)

### 3) High - Playback write operations allowed for any authenticated role

- Previous finding: POST/PUT/DELETE media routes lacked elevated-role enforcement.
- Current status: Fixed
- What changed:
  - Media mutation routes now require system_admin or data_steward.
- Evidence:
  - [repo/backend/internal/router/router.go](repo/backend/internal/router/router.go#L160)
  - [repo/backend/internal/router/router.go](repo/backend/internal/router/router.go#L170)
  - [repo/backend/tests/api/scope_enforcement_test.go](repo/backend/tests/api/scope_enforcement_test.go#L194)

### 4) High - Frontend/backend mismatch for lyrics parse response contract

- Previous finding: Frontend expected lrc/content while backend returned lines only.
- Current status: Fixed
- What changed:
  - Backend parse response now includes lrc field.
  - Frontend still supports reading lrc/content.
- Evidence:
  - [repo/backend/internal/handlers/playback_handler.go](repo/backend/internal/handlers/playback_handler.go#L273)
  - [repo/frontend/src/pages/PlaybackPage.vue](repo/frontend/src/pages/PlaybackPage.vue#L367)
  - [repo/frontend/src/pages/PlaybackPage.vue](repo/frontend/src/pages/PlaybackPage.vue#L368)

### 5) High - Reports frontend adapter/query/response mismatch

- Previous finding: Wrong getRuns call shape, wrong date key names, wrong access field.
- Current status: Fixed
- What changed:
  - Frontend now sends schedule_id, date_from, date_to via getRuns(params).
  - Frontend now checks has_access field.
- Evidence:
  - [repo/frontend/src/pages/ReportsPage.vue](repo/frontend/src/pages/ReportsPage.vue#L368)
  - [repo/frontend/src/pages/ReportsPage.vue](repo/frontend/src/pages/ReportsPage.vue#L369)
  - [repo/frontend/src/pages/ReportsPage.vue](repo/frontend/src/pages/ReportsPage.vue#L374)
  - [repo/frontend/src/pages/ReportsPage.vue](repo/frontend/src/pages/ReportsPage.vue#L399)
  - [repo/frontend/src/api/reports.js](repo/frontend/src/api/reports.js#L27)

### 6) Medium - Frontend reports route roles conflicted with backend authorization

- Previous finding: Frontend allowed standard_user while backend reports routes did not.
- Current status: Fixed
- What changed:
  - Frontend reports route roles now match backend (system_admin, operations_analyst).
- Evidence:
  - [repo/frontend/src/router/index.js](repo/frontend/src/router/index.js#L45)
  - [repo/backend/internal/router/router.go](repo/backend/internal/router/router.go#L194)

### 7) Medium - Scope enforcement report test was non-assertive

- Previous finding: Test logged response without strict pass/fail assertion.
- Current status: Fixed
- What changed:
  - Added strict cross-scope assertions for GET /reports/runs/:id and access-check result.
- Evidence:
  - [repo/backend/tests/api/scope_enforcement_test.go](repo/backend/tests/api/scope_enforcement_test.go#L133)
  - [repo/backend/tests/api/scope_enforcement_test.go](repo/backend/tests/api/scope_enforcement_test.go#L153)
  - [repo/backend/tests/api/scope_enforcement_test.go](repo/backend/tests/api/scope_enforcement_test.go#L164)
  - [repo/backend/tests/api/scope_enforcement_test.go](repo/backend/tests/api/scope_enforcement_test.go#L179)

### 8) Medium - Security regression tests were simulation-heavy

- Previous finding: Critical security tests used simulated routes rather than real handler path.
- Current status: Not Fixed
- Why:
  - The simulation-based pattern is still present in security_regression_test.go.
- Evidence:
  - [repo/backend/tests/api/security_regression_test.go](repo/backend/tests/api/security_regression_test.go#L15)
  - [repo/backend/tests/api/security_regression_test.go](repo/backend/tests/api/security_regression_test.go#L34)

### 9) Low - Frontend compile check in run_tests.sh could not fail

- Previous finding: vue-tsc step used fallback that always passed.
- Current status: Fixed
- What changed:
  - The type-check step now has explicit pass/fail branches and no unconditional success fallback.
- Evidence:
  - [repo/run_tests.sh](repo/run_tests.sh#L35)
  - [repo/run_tests.sh](repo/run_tests.sh#L37)
  - [repo/run_tests.sh](repo/run_tests.sh#L40)

## Conclusion

Most previously reported defects have been addressed in code and route/test wiring. The primary remaining open item is the simulation-heavy nature of security_regression_test.go, which still weakens confidence compared with real-router, DB-backed security regression coverage.
