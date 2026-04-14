# audit_report-1 Fix Check (Static Verification)

Source checked: `.tmp/audit_report-1.md`
Scope: static code review only (no runtime execution, no tests run).

## Verdict
Partial fix. Most previously reported Blocker/High items are fixed, but at least one High issue remains and one integration mismatch remains.

## Finding-by-Finding Status

| Finding ID | Previous Issue (from audit_report-1.md) | Current Status | Evidence |
|---|---|---|---|
| F-001 | Frontend/backend API contract drift | **Partially Fixed** | Major route alignment is now present across org/context/media/reports/security/versions (`repo/frontend/src/api/playback.js:3-40`, `repo/frontend/src/api/reports.js:15-40`, `repo/frontend/src/api/security.js:31-76`, `repo/backend/internal/router/router.go:87-92,160-171,195-205,234-245`). Remaining mismatch: frontend still sends `parentId` camelCase (`repo/frontend/src/api/org.js:7-9`) while backend binds `parent_id` snake_case (`repo/backend/internal/org/org_service.go:20-28`). |
| F-002 | Ingestion/report schedulers declared but not started | **Fixed** | Schedulers are now created, started, and stopped in server lifecycle (`repo/backend/cmd/server/main.go:47-67,96-100`). |
| F-003 | Role taxonomy mismatch (`super_admin` vs `system_admin`) | **Fixed** | DB role enum/seed and RBAC constants use `system_admin` (`repo/backend/migrations/init.sql:21,612-619`; `repo/backend/internal/rbac/rbac.go:12`), and docs match (`repo/README.md:57,219`). |
| F-004 | Audit-log purge path bypassed dual approval | **Fixed** | Security purge endpoints now explicitly reject `audit_logs` and direct callers to dual-approval workflow (`repo/backend/internal/security/security_service.go:472-476,499-503`), while audit delete-request approve/execute routes exist (`repo/backend/internal/router/router.go:215-219`). |
| F-005 | Scope enforcement incomplete (context keys/query scoping) | **Partially Fixed (High remains)** | Middleware now sets trusted scope keys (`repo/backend/internal/auth/auth_middleware.go:108-114`), and analytics/report routes enforce scope context (`repo/backend/internal/router/router.go:174-179,189-194`). However, analytics KPI queries still mostly ignore city/department scope in service-level counts (e.g., `countMasterRecords`, `countIngestionJobs`, `countReportRuns`, `countAuditEvents`) (`repo/backend/internal/analytics/analytics_service.go:215-295`). |
| F-006 | Report format fallback to CSV for non-csv | **Fixed** | Report generation now has explicit CSV/PDF/XLSX branches (`repo/backend/internal/reports/report_service.go:317-324`) with dedicated generators (`:330-461`). |
| F-007 | Tests were mostly mocked and weak for real integration coverage | **Not Fixed** | API tests still rely on in-memory stores and fake auth/mocked routers rather than real handlers + DB-backed flows (`repo/backend/tests/api/helpers_test.go:28-36,78-139`; `repo/backend/tests/api/master_api_test.go:18-36,113-211`; `repo/backend/tests/api/audit_api_test.go:17-49,151-255`; `repo/backend/tests/api/auth_api_test.go:22-24,36-63`). |
| F-008 | CORS wildcard with credentials risk | **Fixed** | CORS now uses configured allowlist with credentials (`repo/backend/internal/router/router.go:32-39`), and documented default is a concrete origin, not wildcard (`repo/backend/internal/config/config.go:66`; `repo/README.md:83`). |

## Residual Risk Summary

1. **High residual risk**: analytics scope is enforced at middleware presence level but not consistently applied in KPI DB queries (`repo/backend/internal/analytics/analytics_service.go:215-295`).
2. **Contract residual**: org node create request still has `parentId` vs `parent_id` mismatch (`repo/frontend/src/api/org.js:7-9`, `repo/backend/internal/org/org_service.go:20-28`).
3. **Test confidence gap**: API suite remains predominantly simulated; limited confidence for true end-to-end behavior (`repo/backend/tests/api/helpers_test.go:28-36,78-139`).
