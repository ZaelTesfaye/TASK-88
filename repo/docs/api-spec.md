# API Specification

Base URL: `/api/v1`
Authentication: JWT Bearer token (except `/auth/login`)
Correlation: Every response includes `X-Correlation-ID` header

Route registration: `backend/internal/router/router.go`

## Error Contract

Every error response follows this schema (implemented in `backend/internal/errors/errors.go`):

```json
{
  "code": "BAD_REQUEST",
  "message": "human-readable description",
  "details": { ... },
  "correlationId": "uuid-string"
}
```

## Status Code Policy

| HTTP Status | Error Code | When Used |
|---|---|---|
| 200 | -- | Successful GET/POST/PUT/PATCH/DELETE |
| 400 | `BAD_REQUEST` | Malformed JSON, missing required fields |
| 401 | `AUTH_REQUIRED` | Missing/invalid/expired token, revoked session |
| 403 | `FORBIDDEN` | Insufficient role, locked account, out-of-scope, non-LAN IP |
| 404 | `NOT_FOUND` | Resource does not exist |
| 409 | `CONFLICT` | Duplicate natural_key, version state conflict |
| 422 | `VALIDATION_ERROR` | Business rule violation (password complexity, invalid entity type, etc.) |
| 500 | `INTERNAL_ERROR` | Unhandled server error, panic recovery |

---

## 1. Auth Endpoints

Handler: `backend/internal/handlers/auth_handler.go`

| Method | Path | Auth | Role Guard |
|---|---|---|---|
| POST | `/api/v1/auth/login` | Public | None |
| POST | `/api/v1/auth/logout` | Required | Any authenticated |
| POST | `/api/v1/auth/refresh` | Required | Any authenticated |

### POST /auth/login

Request: `{ "username": "string", "password": "string" }`

Response (200):
```json
{
  "token": "jwt-access-token",
  "refreshToken": "jwt-refresh-token",
  "user": {
    "id": 1, "username": "admin", "role": "system_admin",
    "city_scope": "*", "department_scope": "*", "status": "active"
  }
}
```

Errors: 400 (bad JSON), 401 (invalid credentials), 403 (locked/inactive)

### POST /auth/logout

Response (200): `{ "message": "logged out successfully" }`

### POST /auth/refresh

Request: `{ "refreshToken": "jwt-refresh-token" }`

Response (200): `{ "token": "new-access", "refreshToken": "new-refresh" }`

---

## 2. Org Endpoints

Handler: `backend/internal/handlers/org_handler.go`

| Method | Path | Role Guard |
|---|---|---|
| GET | `/api/v1/org/tree` | system_admin |
| GET | `/api/v1/org/nodes` | system_admin |
| POST | `/api/v1/org/nodes` | system_admin |
| GET | `/api/v1/org/nodes/:id` | system_admin |
| PUT | `/api/v1/org/nodes/:id` | system_admin |
| DELETE | `/api/v1/org/nodes/:id` | system_admin |
| POST | `/api/v1/context/switch` | system_admin |
| GET | `/api/v1/context/current` | system_admin |

OrgNode schema: `{ parent_id, level_code, level_label, name, city, department, is_active, sort_order }`

---

## 3. Master Data Endpoints

Handler: `backend/internal/handlers/master_handler.go`

| Method | Path | Permission |
|---|---|---|
| GET | `/api/v1/master/:entity` | master_data_view |
| GET | `/api/v1/master/:entity/:id` | master_data_view |
| GET | `/api/v1/master/:entity/:id/history` | master_data_view |
| POST | `/api/v1/master/:entity` | master_data_crud |
| PUT | `/api/v1/master/:entity/:id` | master_data_crud |
| POST | `/api/v1/master/:entity/:id/deactivate` | master_data_crud |

Valid entity types: `sku`, `color`, `size`, `season`, `brand`, `supplier`, `customer`

List query params: `search`, `status` (active/inactive/all), `sort_by`, `sort_order`, `page`, `page_size`

Create request: `{ "natural_key": "SKU001", "payload_json": "{...}" }`

Deactivate request: `{ "reason": "Product discontinued" }`

Entity-specific validation on natural_key:
- SKU: `^[A-Z0-9]{6,20}$`
- Season: `^(SS|FW)[0-9]{4}$`

Duplicate detection: exact match on (entity_type, natural_key); near-match via normalized key comparison.

---

## 4. Version Endpoints

Handler: `backend/internal/handlers/version_handler.go`

| Method | Path | Access |
|---|---|---|
| GET | `/api/v1/versions/:entity` | All 4 roles |
| GET | `/api/v1/versions/:entity/:id` | All 4 roles |
| GET | `/api/v1/versions/:entity/:id/items` | All 4 roles |
| GET | `/api/v1/versions/:entity/:id/diff` | All 4 roles |
| POST | `/api/v1/versions/:entity` | version_draft |
| POST | `/api/v1/versions/:entity/:id/review` | version_draft |
| POST | `/api/v1/versions/:entity/:id/items` | version_draft |
| DELETE | `/api/v1/versions/:entity/:id/items/:itemId` | version_draft |
| POST | `/api/v1/versions/:entity/:id/activate` | system_admin |

Create: `{ "scope_key": "node:3" }` -- auto-increments version_no

Review: `{ "reviewer_id": 2 }` -- reviewer must differ from creator

Activate: archives current active version atomically (row-level lock)

Diff response: `{ "version_a_id", "version_b_id", "added": [], "removed": [], "unchanged": [] }`

---

## 5. Ingestion Endpoints

Handler: `backend/internal/handlers/ingestion_handler.go`

| Method | Path | Access |
|---|---|---|
| GET | `/api/v1/ingestion/sources` | system_admin, operations_analyst |
| POST | `/api/v1/ingestion/sources` | system_admin, operations_analyst |
| GET | `/api/v1/ingestion/sources/:id` | system_admin, operations_analyst |
| PUT | `/api/v1/ingestion/sources/:id` | system_admin, operations_analyst |
| DELETE | `/api/v1/ingestion/sources/:id` | system_admin, operations_analyst |
| GET | `/api/v1/ingestion/jobs` | system_admin, operations_analyst |
| POST | `/api/v1/ingestion/jobs` | system_admin, operations_analyst |
| GET | `/api/v1/ingestion/jobs/:id` | system_admin, operations_analyst |
| POST | `/api/v1/ingestion/jobs/:id/retry` | system_admin, operations_analyst |
| POST | `/api/v1/ingestion/jobs/:id/acknowledge` | system_admin, operations_analyst |
| GET | `/api/v1/ingestion/jobs/:id/checkpoints` | system_admin, operations_analyst |
| GET | `/api/v1/ingestion/jobs/:id/failures` | system_admin, operations_analyst |

Job states: `ready` -> `running` -> `completed` | `retrying` | `failed_awaiting_ack`

Acknowledge request: `{ "reason": "Manual review complete" }` -- required for failed_awaiting_ack jobs

---

## 6. Media (Playback) Endpoints

Handler: `backend/internal/handlers/playback_handler.go`

| Method | Path | Access |
|---|---|---|
| GET | `/api/v1/media` | All authenticated |
| POST | `/api/v1/media` | All authenticated |
| GET | `/api/v1/media/:id` | All authenticated |
| PUT | `/api/v1/media/:id` | All authenticated |
| DELETE | `/api/v1/media/:id` | All authenticated |
| GET | `/api/v1/media/:id/stream` | All authenticated |
| GET | `/api/v1/media/:id/cover` | All authenticated |
| POST | `/api/v1/media/:id/lyrics/parse` | All authenticated |
| GET | `/api/v1/media/:id/lyrics/search` | All authenticated |
| GET | `/api/v1/media/formats/supported` | All authenticated |

Supported formats response: `{ "audio": ["mp3","wav","flac","m4a"], "lyrics": ["lrc"] }`

---

## 7. Analytics Endpoints

Handler: `backend/internal/handlers/analytics_handler.go`

| Method | Path | Access |
|---|---|---|
| GET | `/api/v1/analytics/kpis` | system_admin, operations_analyst |
| GET | `/api/v1/analytics/kpis/definitions` | system_admin, operations_analyst |
| POST | `/api/v1/analytics/kpis/definitions` | system_admin, operations_analyst |
| GET | `/api/v1/analytics/kpis/definitions/:code` | system_admin, operations_analyst |
| PUT | `/api/v1/analytics/kpis/definitions/:code` | system_admin, operations_analyst |
| DELETE | `/api/v1/analytics/kpis/definitions/:code` | system_admin, operations_analyst |
| GET | `/api/v1/analytics/trends` | system_admin, operations_analyst |

KPI result: `{ code, label, value, prev_value, change_percent, trend_direction, unit }`

---

## 8. Report Endpoints

Handler: `backend/internal/handlers/report_handler.go`

| Method | Path | Access |
|---|---|---|
| GET | `/api/v1/reports/schedules` | system_admin, operations_analyst |
| POST | `/api/v1/reports/schedules` | system_admin, operations_analyst |
| GET | `/api/v1/reports/schedules/:id` | system_admin, operations_analyst |
| PATCH | `/api/v1/reports/schedules/:id` | system_admin, operations_analyst |
| DELETE | `/api/v1/reports/schedules/:id` | system_admin, operations_analyst |
| POST | `/api/v1/reports/schedules/:id/trigger` | system_admin, operations_analyst |
| GET | `/api/v1/reports/runs` | system_admin, operations_analyst |
| GET | `/api/v1/reports/runs/:id` | system_admin, operations_analyst |
| GET | `/api/v1/reports/runs/:id/download` | system_admin, operations_analyst |
| GET | `/api/v1/reports/runs/:id/access-check` | system_admin, operations_analyst |

---

## 9. Audit Endpoints

Handler: `backend/internal/handlers/audit_handler.go`

| Method | Path | Access |
|---|---|---|
| GET | `/api/v1/audit/logs` | system_admin |
| GET | `/api/v1/audit/logs/:id` | system_admin |
| GET | `/api/v1/audit/logs/search` | system_admin |
| GET | `/api/v1/audit/delete-requests` | system_admin |
| POST | `/api/v1/audit/delete-requests` | system_admin |
| GET | `/api/v1/audit/delete-requests/:id` | system_admin |
| POST | `/api/v1/audit/delete-requests/:id/approve` | system_admin |
| POST | `/api/v1/audit/delete-requests/:id/execute` | system_admin |

Delete request requires dual approval (approver_one, approver_two).

---

## 10. Security Endpoints

Handler: `backend/internal/handlers/security_handler.go`

| Method | Path | Access |
|---|---|---|
| GET | `/api/v1/security/sensitive-fields` | system_admin |
| POST | `/api/v1/security/sensitive-fields` | system_admin |
| PUT | `/api/v1/security/sensitive-fields/:id` | system_admin |
| DELETE | `/api/v1/security/sensitive-fields/:id` | system_admin |
| GET | `/api/v1/security/keys` | system_admin |
| POST | `/api/v1/security/keys/rotate` | system_admin |
| GET | `/api/v1/security/keys/:id` | system_admin |
| POST | `/api/v1/security/password-reset` | system_admin |
| POST | `/api/v1/security/password-reset/:id/approve` | system_admin |
| GET | `/api/v1/security/password-reset` | system_admin |
| GET | `/api/v1/security/retention-policies` | system_admin |
| POST | `/api/v1/security/retention-policies` | system_admin |
| PUT | `/api/v1/security/retention-policies/:id` | system_admin |
| GET | `/api/v1/security/legal-holds` | system_admin |
| POST | `/api/v1/security/legal-holds` | system_admin |
| POST | `/api/v1/security/legal-holds/:id/release` | system_admin |
| POST | `/api/v1/security/purge-runs/dry-run` | system_admin |
| POST | `/api/v1/security/purge-runs/execute` | system_admin |
| GET | `/api/v1/security/purge-runs` | system_admin |

---

## 11. Integration Endpoints

Handler: `backend/internal/handlers/integration_handler.go`

| Method | Path | Access |
|---|---|---|
| GET | `/api/v1/integrations/endpoints` | system_admin |
| POST | `/api/v1/integrations/endpoints` | system_admin |
| GET | `/api/v1/integrations/endpoints/:id` | system_admin |
| PUT | `/api/v1/integrations/endpoints/:id` | system_admin |
| DELETE | `/api/v1/integrations/endpoints/:id` | system_admin |
| POST | `/api/v1/integrations/endpoints/:id/test` | system_admin |
| GET | `/api/v1/integrations/deliveries` | system_admin |
| GET | `/api/v1/integrations/deliveries/:id` | system_admin |
| POST | `/api/v1/integrations/deliveries/:id/retry` | system_admin |
| GET | `/api/v1/integrations/connectors` | system_admin |
| POST | `/api/v1/integrations/connectors` | system_admin |
| GET | `/api/v1/integrations/connectors/:id` | system_admin |
| PUT | `/api/v1/integrations/connectors/:id` | system_admin |
| DELETE | `/api/v1/integrations/connectors/:id` | system_admin |
| POST | `/api/v1/integrations/connectors/:id/health-check` | system_admin |

URL validation: endpoints must target LAN/private IPs (RFC 1918 + loopback + link-local).

---

## Common Conventions

**Pagination**: All list endpoints accept `page` (1-based, default 1) and `page_size` (default 25-50, max 200). Response includes `total`, `page`, `page_size`, `total_pages`.

**Correlation ID**: Set `X-Correlation-ID` header on requests; auto-generated UUID if absent. Echoed in responses and included in error payloads.

**Timestamps**: All timestamps in RFC 3339 / ISO 8601 format (UTC).
