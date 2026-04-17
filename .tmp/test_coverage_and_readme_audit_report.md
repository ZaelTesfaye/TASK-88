# 1. Test Coverage Audit

## Scope and Method

- Audit mode: static inspection only (no execution of tests/scripts/apps/containers during this audit).
- Project type (declared): `fullstack` (`repo/README.md`, line 1).
- Backend route source of truth: `backend/internal/router/router.go`.

## Backend Endpoint Inventory

Resolved prefix: `/api/v1` for API groups plus standalone `/health`.

Total unique endpoints: **108** (`backend/internal/router/router.go`, `.(GET|POST|PUT|PATCH|DELETE)` occurrences).

| #   | Endpoint (METHOD + PATH)                              |
| --- | ----------------------------------------------------- |
| 1   | GET /health                                           |
| 2   | POST /api/v1/auth/login                               |
| 3   | POST /api/v1/auth/logout                              |
| 4   | POST /api/v1/auth/refresh                             |
| 5   | GET /api/v1/org/tree                                  |
| 6   | GET /api/v1/org/nodes                                 |
| 7   | POST /api/v1/org/nodes                                |
| 8   | GET /api/v1/org/nodes/:id                             |
| 9   | PUT /api/v1/org/nodes/:id                             |
| 10  | DELETE /api/v1/org/nodes/:id                          |
| 11  | POST /api/v1/context/switch                           |
| 12  | GET /api/v1/context/current                           |
| 13  | GET /api/v1/master/:entity                            |
| 14  | GET /api/v1/master/:entity/:id                        |
| 15  | GET /api/v1/master/:entity/:id/history                |
| 16  | POST /api/v1/master/:entity                           |
| 17  | PUT /api/v1/master/:entity/:id                        |
| 18  | POST /api/v1/master/:entity/:id/deactivate            |
| 19  | GET /api/v1/versions/:entity                          |
| 20  | GET /api/v1/versions/:entity/:id                      |
| 21  | GET /api/v1/versions/:entity/:id/items                |
| 22  | GET /api/v1/versions/:entity/:id/diff                 |
| 23  | POST /api/v1/versions/:entity                         |
| 24  | POST /api/v1/versions/:entity/:id/review              |
| 25  | POST /api/v1/versions/:entity/:id/items               |
| 26  | DELETE /api/v1/versions/:entity/:id/items/:itemId     |
| 27  | POST /api/v1/versions/:entity/:id/activate            |
| 28  | GET /api/v1/ingestion/sources                         |
| 29  | POST /api/v1/ingestion/sources                        |
| 30  | GET /api/v1/ingestion/sources/:id                     |
| 31  | PUT /api/v1/ingestion/sources/:id                     |
| 32  | DELETE /api/v1/ingestion/sources/:id                  |
| 33  | GET /api/v1/ingestion/jobs                            |
| 34  | POST /api/v1/ingestion/jobs                           |
| 35  | GET /api/v1/ingestion/jobs/:id                        |
| 36  | POST /api/v1/ingestion/jobs/:id/retry                 |
| 37  | POST /api/v1/ingestion/jobs/:id/acknowledge           |
| 38  | GET /api/v1/ingestion/jobs/:id/checkpoints            |
| 39  | GET /api/v1/ingestion/jobs/:id/failures               |
| 40  | GET /api/v1/media                                     |
| 41  | GET /api/v1/media/:id                                 |
| 42  | GET /api/v1/media/:id/stream                          |
| 43  | GET /api/v1/media/:id/cover                           |
| 44  | GET /api/v1/media/:id/lyrics/search                   |
| 45  | GET /api/v1/media/formats/supported                   |
| 46  | POST /api/v1/media                                    |
| 47  | PUT /api/v1/media/:id                                 |
| 48  | DELETE /api/v1/media/:id                              |
| 49  | POST /api/v1/media/:id/lyrics/parse                   |
| 50  | GET /api/v1/analytics/kpis                            |
| 51  | GET /api/v1/analytics/kpis/definitions                |
| 52  | POST /api/v1/analytics/kpis/definitions               |
| 53  | GET /api/v1/analytics/kpis/definitions/:code          |
| 54  | PUT /api/v1/analytics/kpis/definitions/:code          |
| 55  | DELETE /api/v1/analytics/kpis/definitions/:code       |
| 56  | GET /api/v1/analytics/trends                          |
| 57  | GET /api/v1/reports/schedules                         |
| 58  | POST /api/v1/reports/schedules                        |
| 59  | GET /api/v1/reports/schedules/:id                     |
| 60  | PATCH /api/v1/reports/schedules/:id                   |
| 61  | DELETE /api/v1/reports/schedules/:id                  |
| 62  | POST /api/v1/reports/schedules/:id/trigger            |
| 63  | GET /api/v1/reports/runs                              |
| 64  | GET /api/v1/reports/runs/:id                          |
| 65  | GET /api/v1/reports/runs/:id/download                 |
| 66  | GET /api/v1/reports/runs/:id/access-check             |
| 67  | GET /api/v1/audit/logs                                |
| 68  | GET /api/v1/audit/logs/:id                            |
| 69  | GET /api/v1/audit/logs/search                         |
| 70  | GET /api/v1/audit/delete-requests                     |
| 71  | POST /api/v1/audit/delete-requests                    |
| 72  | GET /api/v1/audit/delete-requests/:id                 |
| 73  | POST /api/v1/audit/delete-requests/:id/approve        |
| 74  | POST /api/v1/audit/delete-requests/:id/execute        |
| 75  | GET /api/v1/security/sensitive-fields                 |
| 76  | POST /api/v1/security/sensitive-fields                |
| 77  | PUT /api/v1/security/sensitive-fields/:id             |
| 78  | DELETE /api/v1/security/sensitive-fields/:id          |
| 79  | GET /api/v1/security/keys                             |
| 80  | POST /api/v1/security/keys/rotate                     |
| 81  | GET /api/v1/security/keys/:id                         |
| 82  | POST /api/v1/security/password-reset                  |
| 83  | POST /api/v1/security/password-reset/:id/approve      |
| 84  | GET /api/v1/security/password-reset                   |
| 85  | GET /api/v1/security/retention-policies               |
| 86  | POST /api/v1/security/retention-policies              |
| 87  | PUT /api/v1/security/retention-policies/:id           |
| 88  | GET /api/v1/security/legal-holds                      |
| 89  | POST /api/v1/security/legal-holds                     |
| 90  | POST /api/v1/security/legal-holds/:id/release         |
| 91  | POST /api/v1/security/purge-runs/dry-run              |
| 92  | POST /api/v1/security/purge-runs/execute              |
| 93  | GET /api/v1/security/purge-runs                       |
| 94  | GET /api/v1/integrations/endpoints                    |
| 95  | POST /api/v1/integrations/endpoints                   |
| 96  | GET /api/v1/integrations/endpoints/:id                |
| 97  | PUT /api/v1/integrations/endpoints/:id                |
| 98  | DELETE /api/v1/integrations/endpoints/:id             |
| 99  | POST /api/v1/integrations/endpoints/:id/test          |
| 100 | GET /api/v1/integrations/deliveries                   |
| 101 | GET /api/v1/integrations/deliveries/:id               |
| 102 | POST /api/v1/integrations/deliveries/:id/retry        |
| 103 | GET /api/v1/integrations/connectors                   |
| 104 | POST /api/v1/integrations/connectors                  |
| 105 | GET /api/v1/integrations/connectors/:id               |
| 106 | PUT /api/v1/integrations/connectors/:id               |
| 107 | DELETE /api/v1/integrations/connectors/:id            |
| 108 | POST /api/v1/integrations/connectors/:id/health-check |

## API Test Mapping Table

Evidence sources for true no-mock HTTP:

- `backend/tests/api/nomock_api_test.go` (integration-tagged, real router + real DB via `skipIfNoDB`, line 33).
- `backend/tests/api/scope_enforcement_test.go`.
- `backend/tests/api/security_regression_test.go`.

| Endpoint                                              | Covered | Test Type         | Test Files                                                                                     | Evidence                                                                                          |
| ----------------------------------------------------- | ------- | ----------------- | ---------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| GET /health                                           | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealHealthEndpoint` (line 1299)                                                              |
| POST /api/v1/auth/login                               | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealAuthLogin` (line 49)                                                                     |
| POST /api/v1/auth/logout                              | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealAuthLogout` (line 94)                                                                    |
| POST /api/v1/auth/refresh                             | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealAuthRefresh` (line 107)                                                                  |
| GET /api/v1/org/tree                                  | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealOrgTree` (line 143)                                                                      |
| GET /api/v1/org/nodes                                 | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealOrgListNodes` (line 159)                                                                 |
| POST /api/v1/org/nodes                                | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealOrgCreateNode` (line 175)                                                                |
| GET /api/v1/org/nodes/:id                             | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealOrgGetNode` (line 191)                                                                   |
| PUT /api/v1/org/nodes/:id                             | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealOrgUpdateNode` (line 218)                                                                |
| DELETE /api/v1/org/nodes/:id                          | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealOrgDeleteNode` (line 234)                                                                |
| POST /api/v1/context/switch                           | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealContextSwitchAndGet` (line 262)                                                          |
| GET /api/v1/context/current                           | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealContextSwitchAndGet` (line 262)                                                          |
| GET /api/v1/master/:entity                            | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealMasterListAndCreate` (line 290)                                                          |
| GET /api/v1/master/:entity/:id                        | yes     | true no-mock HTTP | `backend/tests/api/scope_enforcement_test.go`                                                  | `TestGetMasterRecordCrossScopeDenied` (line 14)                                                   |
| GET /api/v1/master/:entity/:id/history                | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealMasterHistory` (line 333)                                                                |
| POST /api/v1/master/:entity                           | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealMasterListAndCreate` (line 290)                                                          |
| PUT /api/v1/master/:entity/:id                        | yes     | true no-mock HTTP | `backend/tests/api/scope_enforcement_test.go`                                                  | `TestUpdateMasterRecordCrossScopeDenied` (line 55)                                                |
| POST /api/v1/master/:entity/:id/deactivate            | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealMasterDeactivate` (line 318)                                                             |
| GET /api/v1/versions/:entity                          | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealVersionsCRUD` (line 374)                                                                 |
| GET /api/v1/versions/:entity/:id                      | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealVersionsCRUD` (line 374)                                                                 |
| GET /api/v1/versions/:entity/:id/items                | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealVersionsCRUD` (line 374)                                                                 |
| GET /api/v1/versions/:entity/:id/diff                 | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealVersionsCRUD` (line 374)                                                                 |
| POST /api/v1/versions/:entity                         | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealVersionsCRUD` (line 374)                                                                 |
| POST /api/v1/versions/:entity/:id/review              | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealVersionsCRUD` (line 374)                                                                 |
| POST /api/v1/versions/:entity/:id/items               | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealVersionsCRUD` (line 374)                                                                 |
| DELETE /api/v1/versions/:entity/:id/items/:itemId     | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealVersionRemoveItem` (line 1189)                                                           |
| POST /api/v1/versions/:entity/:id/activate            | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealVersionsCRUD` (line 374)                                                                 |
| GET /api/v1/ingestion/sources                         | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIngestionSourcesCRUD` (line 461)                                                         |
| POST /api/v1/ingestion/sources                        | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIngestionSourcesCRUD` (line 461)                                                         |
| GET /api/v1/ingestion/sources/:id                     | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIngestionSourcesCRUD` (line 461)                                                         |
| PUT /api/v1/ingestion/sources/:id                     | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIngestionSourcesCRUD` (line 461)                                                         |
| DELETE /api/v1/ingestion/sources/:id                  | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIngestionSourcesCRUD` (line 461)                                                         |
| GET /api/v1/ingestion/jobs                            | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIngestionJobsLifecycle` (line 502)                                                       |
| POST /api/v1/ingestion/jobs                           | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIngestionJobsLifecycle` (line 502)                                                       |
| GET /api/v1/ingestion/jobs/:id                        | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIngestionJobsLifecycle` (line 502)                                                       |
| POST /api/v1/ingestion/jobs/:id/retry                 | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIngestionJobsLifecycle` (line 502)                                                       |
| POST /api/v1/ingestion/jobs/:id/acknowledge           | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIngestionJobsLifecycle` (line 502)                                                       |
| GET /api/v1/ingestion/jobs/:id/checkpoints            | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIngestionJobsLifecycle` (line 502)                                                       |
| GET /api/v1/ingestion/jobs/:id/failures               | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIngestionJobsLifecycle` (line 502)                                                       |
| GET /api/v1/media                                     | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealMediaEndpoints` (line 593)                                                               |
| GET /api/v1/media/:id                                 | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealMediaEndpoints` (line 593)                                                               |
| GET /api/v1/media/:id/stream                          | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealMediaEndpoints` (line 593)                                                               |
| GET /api/v1/media/:id/cover                           | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealMediaEndpoints` (line 593), `TestRealMediaCoverEndpoint` (line 1273)                     |
| GET /api/v1/media/:id/lyrics/search                   | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealMediaEndpoints` (line 593)                                                               |
| GET /api/v1/media/formats/supported                   | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealMediaEndpoints` (line 593)                                                               |
| POST /api/v1/media                                    | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealMediaEndpoints` (line 593), `TestMediaMutationsRequireElevatedRole` (line 166)           |
| PUT /api/v1/media/:id                                 | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealMediaEndpoints` (line 593)                                                               |
| DELETE /api/v1/media/:id                              | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealMediaEndpoints` (line 593), `TestMediaMutationsRequireElevatedRole` (line 166)           |
| POST /api/v1/media/:id/lyrics/parse                   | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealMediaLyricsParse` (line 1386)                                                            |
| GET /api/v1/analytics/kpis                            | yes     | true no-mock HTTP | `backend/tests/api/scope_enforcement_test.go`, `backend/tests/api/security_regression_test.go` | `TestAnalyticsKPIScopedResults` (line 100), `TestAnalyticsScopeFromAuthContext` (line 64)         |
| GET /api/v1/analytics/kpis/definitions                | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealAnalyticsEndpoints` (line 722)                                                           |
| POST /api/v1/analytics/kpis/definitions               | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealAnalyticsEndpoints` (line 722)                                                           |
| GET /api/v1/analytics/kpis/definitions/:code          | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealAnalyticsEndpoints` (line 722)                                                           |
| PUT /api/v1/analytics/kpis/definitions/:code          | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealAnalyticsEndpoints` (line 722)                                                           |
| DELETE /api/v1/analytics/kpis/definitions/:code       | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealAnalyticsEndpoints` (line 722)                                                           |
| GET /api/v1/analytics/trends                          | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealAnalyticsEndpoints` (line 722)                                                           |
| GET /api/v1/reports/schedules                         | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealReportsLifecycle` (line 787)                                                             |
| POST /api/v1/reports/schedules                        | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealReportsLifecycle` (line 787)                                                             |
| GET /api/v1/reports/schedules/:id                     | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealReportsLifecycle` (line 787)                                                             |
| PATCH /api/v1/reports/schedules/:id                   | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealReportsLifecycle` (line 787)                                                             |
| DELETE /api/v1/reports/schedules/:id                  | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealReportsLifecycle` (line 787)                                                             |
| POST /api/v1/reports/schedules/:id/trigger            | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealReportsLifecycle` (line 787)                                                             |
| GET /api/v1/reports/runs                              | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealReportsLifecycle` (line 787)                                                             |
| GET /api/v1/reports/runs/:id                          | yes     | true no-mock HTTP | `backend/tests/api/scope_enforcement_test.go`                                                  | `TestReportGetRunCrossScopeDenied` (line 123)                                                     |
| GET /api/v1/reports/runs/:id/download                 | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealReportsLifecycle` (line 787)                                                             |
| GET /api/v1/reports/runs/:id/access-check             | yes     | true no-mock HTTP | `backend/tests/api/scope_enforcement_test.go`                                                  | `TestReportAccessCheckCrossScopeDenied` (line 144)                                                |
| GET /api/v1/audit/logs                                | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealAuditEndpoints` (line 924)                                                               |
| GET /api/v1/audit/logs/:id                            | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealAuditEndpoints` (line 924)                                                               |
| GET /api/v1/audit/logs/search                         | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealAuditEndpoints` (line 924)                                                               |
| GET /api/v1/audit/delete-requests                     | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealAuditEndpoints` (line 924)                                                               |
| POST /api/v1/audit/delete-requests                    | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealAuditEndpoints` (line 924)                                                               |
| GET /api/v1/audit/delete-requests/:id                 | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealAuditEndpoints` (line 924)                                                               |
| POST /api/v1/audit/delete-requests/:id/approve        | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealAuditEndpoints` (line 924), `TestRealAuditDeleteRequestExecute` (line 1317)              |
| POST /api/v1/audit/delete-requests/:id/execute        | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealAuditDeleteRequestExecute` (line 1317)                                                   |
| GET /api/v1/security/sensitive-fields                 | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealSecurityEndpoints` (line 987)                                                            |
| POST /api/v1/security/sensitive-fields                | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealSecurityEndpoints` (line 987)                                                            |
| PUT /api/v1/security/sensitive-fields/:id             | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealSecurityEndpoints` (line 987)                                                            |
| DELETE /api/v1/security/sensitive-fields/:id          | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealSecurityEndpoints` (line 987)                                                            |
| GET /api/v1/security/keys                             | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealSecurityEndpoints` (line 987)                                                            |
| POST /api/v1/security/keys/rotate                     | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealSecurityEndpoints` (line 987)                                                            |
| GET /api/v1/security/keys/:id                         | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealSecurityEndpoints` (line 987)                                                            |
| POST /api/v1/security/password-reset                  | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealSecurityEndpoints` (line 987)                                                            |
| POST /api/v1/security/password-reset/:id/approve      | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealSecurityEndpoints` (line 987)                                                            |
| GET /api/v1/security/password-reset                   | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealSecurityEndpoints` (line 987)                                                            |
| GET /api/v1/security/retention-policies               | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealSecurityEndpoints` (line 987)                                                            |
| POST /api/v1/security/retention-policies              | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealSecurityEndpoints` (line 987)                                                            |
| PUT /api/v1/security/retention-policies/:id           | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealSecurityEndpoints` (line 987)                                                            |
| GET /api/v1/security/legal-holds                      | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealSecurityEndpoints` (line 987)                                                            |
| POST /api/v1/security/legal-holds                     | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealSecurityEndpoints` (line 987)                                                            |
| POST /api/v1/security/legal-holds/:id/release         | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealSecurityEndpoints` (line 987)                                                            |
| POST /api/v1/security/purge-runs/dry-run              | yes     | true no-mock HTTP | `backend/tests/api/security_regression_test.go`                                                | `TestAuditLogDryRunViaSecurityEndpointRejected` (line 31)                                         |
| POST /api/v1/security/purge-runs/execute              | yes     | true no-mock HTTP | `backend/tests/api/security_regression_test.go`                                                | `TestAuditLogPurgeViaSecurityEndpointRejected` (line 15), `TestNonAuditLogPurgeAllowed` (line 50) |
| GET /api/v1/security/purge-runs                       | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealSecurityEndpoints` (line 987)                                                            |
| GET /api/v1/integrations/endpoints                    | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIntegrationEndpoints` (line 1101)                                                        |
| POST /api/v1/integrations/endpoints                   | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIntegrationEndpoints` (line 1101)                                                        |
| GET /api/v1/integrations/endpoints/:id                | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIntegrationEndpoints` (line 1101)                                                        |
| PUT /api/v1/integrations/endpoints/:id                | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIntegrationEndpoints` (line 1101)                                                        |
| DELETE /api/v1/integrations/endpoints/:id             | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIntegrationEndpoints` (line 1101)                                                        |
| POST /api/v1/integrations/endpoints/:id/test          | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIntegrationEndpoints` (line 1101)                                                        |
| GET /api/v1/integrations/deliveries                   | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIntegrationEndpoints` (line 1101)                                                        |
| GET /api/v1/integrations/deliveries/:id               | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIntegrationGetDeliveryByID` (line 1239)                                                  |
| POST /api/v1/integrations/deliveries/:id/retry        | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIntegrationDeliveryRetry` (line 1361)                                                    |
| GET /api/v1/integrations/connectors                   | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIntegrationEndpoints` (line 1101)                                                        |
| POST /api/v1/integrations/connectors                  | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIntegrationEndpoints` (line 1101)                                                        |
| GET /api/v1/integrations/connectors/:id               | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIntegrationEndpoints` (line 1101)                                                        |
| PUT /api/v1/integrations/connectors/:id               | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIntegrationEndpoints` (line 1101)                                                        |
| DELETE /api/v1/integrations/connectors/:id            | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIntegrationEndpoints` (line 1101)                                                        |
| POST /api/v1/integrations/connectors/:id/health-check | yes     | true no-mock HTTP | `backend/tests/api/nomock_api_test.go`                                                         | `TestRealIntegrationEndpoints` (line 1101)                                                        |

## API Test Classification

1. True No-Mock HTTP

- `backend/tests/api/nomock_api_test.go`
- `backend/tests/api/scope_enforcement_test.go`
- `backend/tests/api/security_regression_test.go`
- Evidence: real production router + real DB (`skipIfNoDB`, `realRouter`, `loginAndGetToken`) in `backend/tests/api/nomock_api_test.go:33`, `backend/tests/api/integration_helpers_test.go:49`.

2. HTTP with Mocking / Dependency Substitution

- `backend/tests/contract/route_contract_test.go` and `backend/tests/unit/router_test.go` use `router.SetupRouter(cfg, nil)` (`route_contract_test.go:166`, `router_test.go:18`), so dependencies are not real DB-backed execution paths.
- These are HTTP/router-contract checks, not full no-mock API execution.

3. Non-HTTP (unit/integration without HTTP)

- `backend/tests/api/db_connector_test.go` exercises `ingestion.NewDatabaseConnector(...).Pull/HealthCheck` directly (`db_connector_test.go:67`, `:123`, `:149`), no API routing.
- `backend/tests/unit/*` package-level tests (auth/rbac/models/encryption/parser/scheduler/etc.) are predominantly non-HTTP unit tests.

## Mock Detection

### Backend API test layer

- No `jest.mock` / `vi.mock` / `sinon.stub` found in `backend/tests/api`.
- However, dependency substitution exists in router/contract tests via nil DB:
  - `backend/tests/contract/route_contract_test.go:166`
  - `backend/tests/unit/router_test.go:18`

### Frontend test layer (explicit mocks present)

- Extensive `vi.mock` in frontend unit/component tests, e.g.:
  - `frontend/src/tests/unit/app.test.js:6`
  - `frontend/src/tests/unit/org-tree.test.js:6`
  - `frontend/src/tests/component/http_integration.test.js:17`
  - `frontend/src/tests/component/api-contract.test.js:4`

## Coverage Summary

- Total endpoints: **108**
- Endpoints with HTTP tests: **108**
- Endpoints with true no-mock HTTP tests: **108**
- HTTP coverage: **100%**
- True API coverage: **100%**

## Unit Test Analysis

### Backend Unit Tests

- Test files (sample): `backend/tests/unit/auth_test.go`, `.../router_test.go`, `.../rbac_test.go`, `.../services_test.go`, `.../scheduler_test.go`, `.../security_handler_test.go`.
- Modules covered:
  - controllers/handlers: org, master, version, report, playback, ingestion, integration, audit, security
  - services/domain: analytics, ingestion, connector, retention, scheduler, auth/password, encryption
  - middleware/auth/rbac: middleware, rbac, scope, router
  - models/errors/config/database helpers
- Important backend modules not explicitly tested (direct evidence absent in `backend/tests`):
  - `backend/internal/logging/*`
  - route-registration helper functions inside several handler files are indirectly covered but not independently asserted as units.

### Frontend Unit Tests (STRICT REQUIREMENT)

- Frontend test files: present under `frontend/src/tests/unit` and `frontend/src/tests/component` (e.g., `org-tree.test.js`, `security.test.js`, `reports.test.js`, `components.test.js`).
- Framework/tools detected:
  - Vitest (`frontend/vitest.config.js`)
  - Vue Test Utils (`frontend/src/tests/unit/org-tree.test.js:2`)
  - Playwright E2E (`frontend/playwright.config.js`, `frontend/src/tests/e2e/*.spec.js`)
- Components/modules covered:
  - Pages: Login, OrgTree, MasterData, Playback, Analytics, Ingestion, Reports, SecurityAdmin
  - Stores: auth, context
  - API adapters: auth/org/master/versions/ingestion/playback/analytics/reports/security/audit/integrations
  - Common UI components: AppButton, AppChip, AppDialog, AppTable, AppInput, AppFileUpload, AppBreadcrumb, AppToast, AppSelect, AppLoadingState, AppErrorState, AppEmptyState
- Important frontend modules NOT tested (direct dedicated test not found):
  - No dedicated unit test file for `frontend/src/router/index.js` implementation internals beyond guard behavior tests (`router.guards.test.js` covers behavior, not full route map completeness).

**Frontend unit tests: PRESENT**

### Cross-Layer Observation

- Testing is balanced across backend and frontend.
- No backend-heavy/frontend-empty imbalance detected.

## API Observability Check

- Strong areas: many API tests assert status plus key JSON fields (tokens, role, message, kpis, has_access, etc.).
- Weak areas:
  - Some tests accept broad status ranges (`200/404/500`) and do limited payload assertions (e.g., media cover/download tolerant paths in `nomock_api_test.go`).
  - Contract/router tests often validate registration or unauthenticated middleware response only; they do not prove handler-level request/response behavior.

## Test Quality & Sufficiency

- Success paths: broad coverage across all endpoint groups.
- Failure/negative paths: present (unauthorized, forbidden, invalid input, scope violations, wrong creds).
- Edge cases: partial (scope mismatch, no context, purge policy constraints, missing resources).
- Validation/auth/permissions: explicitly tested in multiple suites.
- Integration boundaries:
  - Strong at API layer for integration-tagged tests.
  - Mixed for contract/router tests due nil DB substitution.
- Assertions depth: mixed; many meaningful assertions, but some shallow status-only checks remain.

### run_tests.sh check

- Docker-based execution: **YES** (uses `docker run` wrappers).
- Local dependency/runtime install required on host: **NO** (installs happen inside containers).

## End-to-End Expectations (Fullstack)

- Real FE?BE E2E tests are present (`frontend/src/tests/e2e/*.spec.js`, Playwright config).
- Therefore, expected fullstack E2E layer exists.

## Tests Check

- Static evidence supports complete endpoint coverage with integration-tagged no-mock API tests.
- Risk: these no-mock tests are build-tag/DB-gated and not part of default run unless integration tag + DB env are provided.

## Test Coverage Score (0�100)

**91/100**

## Score Rationale

- - Full endpoint inventory mapped and covered.
- - True no-mock API suite exists with real router + real DB.
- - Frontend and backend unit coverage breadth is high.
- - Several assertions are permissive or shallow for specific endpoints.
- - Part of coverage relies on integration-tag execution path not default in standard test invocation.

## Key Gaps

- Permissive assertions in a subset of no-mock tests reduce confidence in response-contract strictness.
- Router/contract HTTP tests with nil DB should not be treated as true no-mock business-flow tests.
- `backend/internal/logging` lacks direct test evidence.

## Confidence & Assumptions

- Confidence: **high** for route inventory and test-file mapping; **medium-high** for handler-reach inference where only status assertions are present.
- Assumption: parameterized concrete test paths (e.g., `/api/v1/master/sku/1`) are valid coverage for normalized route templates (e.g., `/api/v1/master/:entity/:id`).

---

# 2. README Audit

## README Location Check

- Required file exists: `repo/README.md`.

## Hard Gates

### Formatting

- PASS: Markdown is structured and readable.

### Startup Instructions (Fullstack)

- **FAIL (strict gate)**: required literal command `docker-compose up` is not present.
- Found instead: `docker compose up --build` (`repo/README.md`, Quick Start).

### Access Method

- PASS: frontend URL and backend URL are documented (`http://localhost:3000`, `http://localhost:8080`).

### Verification Method

- PASS: includes curl API verification and UI verification flow.

### Environment Rules (No host runtime installs)

- PASS in README instructions: explicitly says Docker-contained and no host runtime installs.

### Demo Credentials (auth exists)

- PASS: includes usernames, password, and all roles (`system_admin`, `data_steward`, `operations_analyst`, `standard_user`).

## Engineering Quality

- Tech stack clarity: strong.
- Architecture explanation: strong (diagram + component responsibilities).
- Testing instructions: strong and detailed.
- Security/roles documentation: strong.
- Operational workflows/troubleshooting: strong.
- Presentation quality: strong.

## High Priority Issues

- Required strict startup literal mismatch: missing `docker-compose up` string.

## Medium Priority Issues

- `run_tests.sh` details mention `npm install` and `go mod download` internally; README should explicitly clarify these are container-internal to avoid policy ambiguity.

## Low Priority Issues

- Minor command consistency (`docker compose` vs `docker-compose`) appears in different places (Playwright comment uses hyphenated style, quick start uses spaced style).

## Hard Gate Failures

- Missing exact required startup instruction token: `docker-compose up`.

## README Verdict

**FAIL**

Reason: strict hard-gate violation on required startup command text.
