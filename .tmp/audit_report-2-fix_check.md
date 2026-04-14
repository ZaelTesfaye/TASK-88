# Fix Check Report for audit_report-2.md (Static Re-Inspection)

Date: 2026-04-14
Mode: Static only (no runtime, no tests, no Docker)
Source baseline reviewed: .tmp/audit_report-2.md

## Overall Result

- Fixed: 8
- Partially Fixed: 1
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
- Current status: Partially Fixed (scoped resolution)
- Why:
  - security_regression_test.go now uses real production router + DB-backed setup and explicitly documents no simulated routes.
  - However, the broader API test suite still contains simulation-heavy patterns: helpers_test.go uses fakeAuthMiddleware and testRouter; master_api_test.go uses in-memory master record store.
  - Resolution is scoped to security_regression scope only; suite-level test realism remains mixed.
- Evidence (real-router integration):
  - [repo/backend/tests/api/security_regression_test.go](repo/backend/tests/api/security_regression_test.go#L12)
  - [repo/backend/tests/api/security_regression_test.go](repo/backend/tests/api/security_regression_test.go#L14)
  - [repo/backend/tests/api/security_regression_test.go](repo/backend/tests/api/security_regression_test.go#L31)
  - [repo/backend/tests/api/security_regression_test.go](repo/backend/tests/api/security_regression_test.go#L57)
  - [repo/backend/tests/api/integration_helpers_test.go](repo/backend/tests/api/integration_helpers_test.go) (real DB + router.SetupRouter)
- Evidence (remaining simulation):
  - [repo/backend/tests/api/helpers_test.go](repo/backend/tests/api/helpers_test.go) (fakeAuthMiddleware, testRouter)
  - [repo/backend/tests/api/master_api_test.go](repo/backend/tests/api/master_api_test.go) (in-memory store)
- Migration plan:
  - Migrate remaining simulated API tests (helpers_test.go patterns, master_api_test.go in-memory store) to use integration_helpers_test.go real-router integration pattern.
  - Target: suite-level consistency on DB-backed + real-router pattern for all sensitive (auth, scope, RBAC) test cases.

### 9) Low - Frontend compile check in run_tests.sh could not fail

- Previous status: Fixed
- Current status: Fixed
- Evidence:
  - [repo/run_tests.sh](repo/run_tests.sh#L39)
  - [repo/run_tests.sh](repo/run_tests.sh#L43)

## Traceability: Baseline Finding → Re-Validation Status

| Finding ID | Title                                                                            | Baseline Status (audit_report-2.md)  | Current Re-Validation    | Key Evidence Artifacts                                                                                                                                                         |
| ---------- | -------------------------------------------------------------------------------- | ------------------------------------ | ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1          | Report object-level authorization missing on list/get endpoints                  | Partial Pass                         | Fixed                    | report_handler.go:209,230-231,243; scope_enforcement_test.go:140,168                                                                                                           |
| 2          | Master-data scope can degrade to broad access when context assignment is missing | Partial Pass                         | Fixed                    | master_handler.go:389,404; master_service.go:174-175                                                                                                                           |
| 3          | Playback write operations allowed for any authenticated role                     | Partial Pass                         | Fixed                    | router.go:170-172; scope_enforcement_test.go:270                                                                                                                               |
| 4          | Frontend/backend mismatch for lyrics parse response contract                     | Partial Pass                         | Fixed                    | playback_handler.go:270; PlaybackPage.vue:368; contract_test.go:260                                                                                                            |
| 5          | Reports frontend adapter/query/response mismatch                                 | Partial Pass                         | Fixed                    | ReportsPage.vue:368-369,374,399; reports.js:28; contract_test.go:297,325                                                                                                       |
| 6          | Frontend reports route roles conflicted with backend authorization               | Partial Pass                         | Fixed                    | router.js:42,45; router.go:194                                                                                                                                                 |
| 7          | Scope enforcement report test was non-assertive                                  | Partial Pass                         | Fixed                    | scope_enforcement_test.go:140,168                                                                                                                                              |
| 8          | Security regression tests were simulation-heavy                                  | Partial Pass (Not Fixed in baseline) | Partially Fixed (scoped) | security_regression_test.go:12,14,31,57; integration_helpers_test.go (real DB). Remaining: helpers_test.go (fakeAuth), master_api_test.go (in-memory). Migration plan tracked. |
| 9          | Frontend compile check in run_tests.sh could not fail                            | Partial Pass                         | Fixed                    | run_tests.sh:39,43                                                                                                                                                             |

## Conclusion

Re-validation scope: 9 findings from audit_report-2.md were systematically re-inspected against current codebase state (static review only, no runtime execution).

Closure status:

- 8 findings show evidence of full remediation in code and/or tests.
- 1 finding (Finding #8: API test realism) shows **scoped remediation**: security_regression_test.go successfully migrated to real-router/DB integration, but suite-level simulation patterns persist in helpers_test.go and master_api_test.go.

Suite-level test realism gap: API test coverage remains mixed—security-regression scope resolved, but remaining simulated tests (fakeAuthMiddleware, in-memory stores) should migrate to integration pattern for consistency. A migration plan has been documented in Finding #8.

Further closure validation would require runtime execution and suite-wide verification that all sensitive (auth, scope, RBAC) test cases use real-router + DB-backed patterns.
