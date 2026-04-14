# API Specification

## Overview

All API endpoints are accessible at `http://localhost:8080/api/v1` (or your backend host).

### Authentication Model

All protected endpoints require a valid JWT token in the `Authorization` header:

```
Authorization: Bearer <jwt_token>
```

**Token Acquisition:**
1. POST `/api/v1/auth/login` with username and password → receive JWT access token
2. Include token in all subsequent requests
3. On expiration, POST `/api/v1/auth/refresh` with valid token → receive new token
4. POST `/api/v1/auth/logout` to revoke session

**Token Expiration:**
- **Idle timeout:** 30 minutes of inactivity
- **Absolute timeout:** 24 hours from token creation
- Session tracked in `sessions` table for revocation checking

### Standard Error Response Format

All error responses follow this structure:

```json
{
  "code": "ERROR_CODE",
  "message": "Human-readable error description",
  "details": null,
  "correlationId": "550e8400-e29b-41d4-a716-446655440000"
}
```

**HTTP Status Codes Used:**
- `200 OK` — Request successful
- `201 Created` — Resource created successfully
- `400 Bad Request` — Invalid request body or parameters
- `401 Unauthorized` — Missing or invalid authentication token
- `403 Forbidden` — Authenticated but lacks required role/permission
- `404 Not Found` — Resource does not exist
- `409 Conflict` — Resource already exists or state conflict
- `422 Unprocessable Entity` — Business logic validation failed (e.g., invalid state transition)
- `500 Internal Server Error` — Unexpected server error

---

## Auth Endpoints

### POST /auth/login

**Public endpoint (no authentication required)**

Authenticate with username and password to receive a JWT access token.

**Request Headers:**
```
Content-Type: application/json
```

**Request Body:**
```json
{
  "username": "string (required)",
  "password": "string (required)"
}
```

**Response (200 OK):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresAt": "2026-04-14T12:30:00Z",
  "user": {
    "id": 1,
    "username": "admin",
    "role": "system_admin",
    "city_scope": "*",
    "department_scope": "*",
    "email": "admin@example.com"
  }
}
```

**Response (401 Unauthorized):**
```json
{
  "code": "AUTH_REQUIRED",
  "message": "invalid username or password",
  "details": null,
  "correlationId": "..."
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "Admin@12345678"
  }'
```

---

### POST /auth/logout

**Protected endpoint (authentication required)**

Revoke the current JWT session, logging out the user.

**Request Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:** (empty)

**Response (200 OK):**
```json
{
  "message": "logout successful"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer <token>"
```

---

### POST /auth/refresh

**Protected endpoint (authentication required)**

Refresh an expiring JWT token to extend the session.

**Request Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:** (empty)

**Response (200 OK):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresAt": "2026-04-14T12:30:00Z"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Authorization: Bearer <token>"
```

---

## Organization (Org) Endpoints

**Authorization Required:** `system_admin` role only

### GET /org/tree

Fetch the organization hierarchy as a nested tree structure.

**Request Headers:**
```
Authorization: Bearer <token>
```

**Query Parameters:** None

**Response (200 OK):**
```json
{
  "id": 1,
  "level_code": "root",
  "level_label": "Organization",
  "name": "Global",
  "city": "*",
  "department": "*",
  "children": [
    {
      "id": 2,
      "level_code": "region",
      "level_label": "Region",
      "name": "North America",
      "city": "NYC",
      "department": "*",
      "children": [
        {
          "id": 3,
          "level_code": "department",
          "level_label": "Department",
          "name": "Finance",
          "city": "NYC",
          "department": "Finance",
          "children": []
        }
      ]
    }
  ]
}
```

**Example:**
```bash
curl -X GET http://localhost:8080/api/v1/org/tree \
  -H "Authorization: Bearer <token>"
```

---

### GET /org/nodes

List all organization nodes with pagination.

**Query Parameters:**
- `page` (optional, default: 1) — Page number
- `page_size` (optional, default: 50) — Items per page
- `level` (optional) — Filter by level_code (e.g., "region", "department")

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 1,
      "level_code": "root",
      "level_label": "Organization",
      "name": "Global",
      "city": "*",
      "department": "*",
      "is_active": true,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    },
    {
      "id": 2,
      "level_code": "region",
      "level_label": "Region",
      "name": "North America",
      "city": "NYC",
      "department": "*",
      "is_active": true,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ],
  "total": 42,
  "page": 1,
  "page_size": 50
}
```

---

### POST /org/nodes

Create a new organization node.

**Request Body:**
```json
{
  "parent_id": 1,
  "level_code": "string (required, e.g., 'department')",
  "level_label": "string (required, e.g., 'Department')",
  "name": "string (required)",
  "city": "string (optional)",
  "department": "string (optional)"
}
```

**Response (201 Created):**
```json
{
  "id": 5,
  "parent_id": 1,
  "level_code": "department",
  "level_label": "Department",
  "name": "Sales",
  "city": "NYC",
  "department": "Sales",
  "is_active": true,
  "created_at": "2026-04-14T10:00:00Z",
  "updated_at": "2026-04-14T10:00:00Z"
}
```

---

### GET /org/nodes/:id

Retrieve a specific organization node by ID.

**Path Parameters:**
- `id` (required) — Node ID

**Response (200 OK):**
```json
{
  "id": 2,
  "level_code": "region",
  "level_label": "Region",
  "name": "North America",
  "city": "NYC",
  "department": "*",
  "is_active": true,
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

---

### PUT /org/nodes/:id

Update an organization node.

**Request Body:**
```json
{
  "name": "string (optional)",
  "city": "string (optional)",
  "department": "string (optional)"
}
```

**Response (200 OK):** (updated node object)

---

### DELETE /org/nodes/:id

Delete an organization node (soft delete via is_active flag).

**Response (204 No Content)**

---

### POST /context/switch

Switch the current user's org context (scope).

**Request Body:**
```json
{
  "org_node_id": 3
}
```

**Response (200 OK):**
```json
{
  "message": "context switched successfully",
  "node_id": 3,
  "city_scope": "NYC",
  "dept_scope": "Sales"
}
```

---

### GET /context/current

Get the current user's active org context.

**Response (200 OK):**
```json
{
  "user_id": 1,
  "node_id": 3,
  "city_scope": "NYC",
  "dept_scope": "Sales",
  "node_name": "Sales Department"
}
```

---

## Master Data Endpoints

### GET /master/:entity

List master records of a specific entity type (e.g., "sku", "customer", "location").

**Authorization Required:** `master_data_view` permission

**Path Parameters:**
- `entity` (required) — Entity type (e.g., "sku", "customer")

**Query Parameters:**
- `page` (optional, default: 1)
- `page_size` (optional, default: 50)
- `search` (optional) — Search in natural_key or payload_json
- `status` (optional) — Filter by status: "active", "inactive", "all"
- `sort_by` (optional) — Sort column: "natural_key", "status", "created_at", "updated_at"
- `sort_order` (optional) — "asc" or "desc"

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 101,
      "entity_type": "sku",
      "natural_key": "SKU-001",
      "status": "active",
      "payload_json": {
        "description": "Product A",
        "category": "Electronics",
        "price": 99.99
      },
      "created_by": 1,
      "created_at": "2026-01-15T10:00:00Z",
      "updated_at": "2026-04-14T14:30:00Z"
    }
  ],
  "total": 1234,
  "page": 1,
  "page_size": 50,
  "total_pages": 25
}
```

---

### GET /master/:entity/:id

Retrieve a single master record by ID and entity type.

**Path Parameters:**
- `entity` (required)
- `id` (required)

**Response (200 OK):**
```json
{
  "id": 101,
  "entity_type": "sku",
  "natural_key": "SKU-001",
  "status": "active",
  "payload_json": {
    "description": "Product A",
    "category": "Electronics"
  },
  "created_by": 1,
  "created_at": "2026-01-15T10:00:00Z",
  "updated_at": "2026-04-14T14:30:00Z"
}
```

---

### POST /master/:entity

Create a new master record (in draft, requires version activation).

**Authorization Required:** `master_data_crud` permission

**Request Body:**
```json
{
  "natural_key": "string (required, unique per entity type)",
  "payload_json": {
    "field1": "value1",
    "field2": "value2"
  }
}
```

**Response (201 Created):**
```json
{
  "id": 102,
  "entity_type": "sku",
  "natural_key": "SKU-002",
  "status": "active",
  "payload_json": { ... },
  "created_by": 1,
  "created_at": "2026-04-14T10:00:00Z",
  "updated_at": "2026-04-14T10:00:00Z"
}
```

---

### PUT /master/:entity/:id

Update an existing master record (creates new version if needed).

**Authorization Required:** `master_data_crud` permission

**Request Body:**
```json
{
  "payload_json": {
    "field1": "new_value1",
    "field2": "new_value2"
  }
}
```

**Response (200 OK):** (updated record)

---

### POST /master/:entity/:id/deactivate

Deactivate a master record, making it unavailable for selection.

**Request Body:**
```json
{
  "reason": "string (optional, reason for deactivation)"
}
```

**Response (200 OK):**
```json
{
  "id": 101,
  "status": "inactive",
  "deactivation_reason": "replaced by SKU-003"
}
```

---

### GET /master/:entity/:id/history

Retrieve the history (deactivation events) for a master record.

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": 1,
      "record_id": 101,
      "reason": "replaced by SKU-003",
      "actor_user_id": 2,
      "deactivated_at": "2026-04-14T10:00:00Z"
    }
  ]
}
```

---

## Version Control Endpoints

**Authorization Required:** Varies by operation (view, draft, activate)

### GET /versions/:entity

List all versions for an entity type.

**Query Parameters:**
- `page`, `page_size` (optional)

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 1,
      "entity_type": "sku",
      "scope_key": "node:3",
      "version_no": 1,
      "state": "draft",
      "created_by": 1,
      "created_at": "2026-04-14T09:00:00Z"
    },
    {
      "id": 2,
      "entity_type": "sku",
      "scope_key": "node:3",
      "version_no": 2,
      "state": "active",
      "created_by": 1,
      "created_at": "2026-04-14T10:00:00Z"
    }
  ],
  "total": 42
}
```

---

### GET /versions/:entity/:id

Retrieve a specific version by ID.

**Response (200 OK):**
```json
{
  "id": 2,
  "entity_type": "sku",
  "scope_key": "node:3",
  "version_no": 2,
  "state": "active",
  "created_by": 1,
  "created_at": "2026-04-14T10:00:00Z",
  "activated_at": "2026-04-14T10:15:00Z"
}
```

---

### POST /versions/:entity

Create a new version (draft).

**Authorization Required:** `version_draft` permission

**Request Body:**
```json
{
  "scope_key": "node:3"
}
```

**Response (201 Created):**
```json
{
  "id": 3,
  "entity_type": "sku",
  "scope_key": "node:3",
  "version_no": 3,
  "state": "draft",
  "created_by": 1,
  "created_at": "2026-04-14T10:00:00Z"
}
```

---

### POST /versions/:entity/:id/items

Add items (master records) to a version.

**Request Body:**
```json
{
  "master_record_id": 101
}
```

**Response (201 Created)**

---

### DELETE /versions/:entity/:id/items/:itemId

Remove an item from a version.

**Response (204 No Content)**

---

### POST /versions/:entity/:id/review

Submit a version for review.

**Request Body:**
```json
{
  "comment": "string (optional)"
}
```

**Response (200 OK)**

---

### POST /versions/:entity/:id/activate

Activate a reviewed version (only system_admin).

**Response (200 OK)**
```json
{
  "state": "active",
  "activated_at": "2026-04-14T10:15:00Z"
}
```

---

### GET /versions/:entity/:id/diff

Compare two versions (shows added/removed items).

**Query Parameters:**
- `other_id` (optional) — ID of version to compare against

**Response (200 OK):**
```json
{
  "added": [101, 102],
  "removed": [103],
  "unchanged": [104, 105]
}
```

---

## Media / Playback Endpoints

### GET /media

List all media assets (accessible to all authenticated users).

**Query Parameters:**
- `page`, `page_size` (optional)
- `search` (optional) — Search in filename or tags

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 1,
      "filename": "song1.mp3",
      "artist": "Artist Name",
      "title": "Song Title",
      "duration_seconds": 240,
      "format": "mp3",
      "file_size_bytes": 5242880,
      "tags": ["pop", "2024"],
      "has_lyrics": true,
      "created_at": "2026-01-01T00:00:00Z"
    }
  ],
  "total": 500
}
```

---

### GET /media/:id

Retrieve metadata for a specific media asset.

**Response (200 OK):**
```json
{
  "id": 1,
  "filename": "song1.mp3",
  "artist": "Artist Name",
  "title": "Song Title",
  "duration_seconds": 240,
  "format": "mp3",
  "file_size_bytes": 5242880,
  "created_at": "2026-01-01T00:00:00Z"
}
```

---

### GET /media/:id/stream

Stream audio file content (accessible to all authenticated users).

**Response:** Binary audio file with `Content-Type: audio/mpeg` or appropriate format

---

### GET /media/:id/cover

Retrieve cover art image for media asset.

**Response:** Binary image file (Jpeg/PNG)

---

### POST /media

Upload a new media asset.

**Authorization Required:** `system_admin` or `data_steward` role

**Request Headers:**
```
Content-Type: multipart/form-data
Authorization: Bearer <token>
```

**Request Body:**
```
file: <binary file data>
filename: string (optional, defaults to uploaded filename)
artist: string (optional)
title: string (optional)
tags: string (optional, comma-separated)
```

**Response (201 Created):**
```json
{
  "id": 2,
  "filename": "song2.mp3",
  "file_size_bytes": 4194304,
  "created_at": "2026-04-14T10:00:00Z"
}
```

---

### PUT /media/:id

Update media asset metadata.

**Authorization Required:** `system_admin` or `data_steward` role

**Request Body:**
```json
{
  "artist": "string (optional)",
  "title": "string (optional)",
  "tags": ["string"] (optional)
}
```

**Response (200 OK)**

---

### DELETE /media/:id

Delete a media asset.

**Authorization Required:** `system_admin` or `data_steward` role

**Response (204 No Content)**

---

### POST /media/:id/lyrics/parse

Parse LRC (lyrics) format and return structured data.

**Authorization Required:** `system_admin` or `data_steward` role

**Request Body:**
```json
{
  "lrc": "[00:00.00]Hello world\n[00:02.50]How are you?"
}
```

**Response (200 OK):**
```json
{
  "status": "success",
  "line_count": 2,
  "lines": [
    {
      "time": 0,
      "text": "Hello world"
    },
    {
      "time": 2.5,
      "text": "How are you?"
    }
  ],
  "lrc": "[00:00.00]Hello world\n[00:02.50]How are you?"
}
```

---

### GET /media/:id/lyrics/search

Search within lyrics for a media asset.

**Query Parameters:**
- `q` (required) — Search query

**Response (200 OK):**
```json
{
  "matches": [
    {
      "time": 0,
      "text": "Hello world",
      "line_number": 1
    }
  ],
  "total": 1
}
```

---

### GET /media/formats/supported

List supported audio formats.

**Response (200 OK):**
```json
{
  "formats": ["mp3", "wav", "flac", "m4a"]
}
```

---

## Analytics Endpoints

**Authorization Required:** `system_admin` or `operations_analyst` role + scope enforcement

### GET /analytics/kpis

Retrieve KPI values for the user's scope.

**Query Parameters:**
- `date_from` (optional, YYYY-MM-DD)
- `date_to` (optional, YYYY-MM-DD)
- `kpi_codes` (optional, comma-separated list of KPI codes)

**Response (200 OK):**
```json
{
  "kpis": [
    {
      "code": "sales_revenue",
      "name": "Sales Revenue",
      "value": 150000.50,
      "unit": "USD",
      "period": "2026-04",
      "timestamp": "2026-04-14T14:30:00Z"
    },
    {
      "code": "customer_count",
      "name": "Active Customers",
      "value": 245,
      "unit": "count",
      "period": "2026-04",
      "timestamp": "2026-04-14T14:30:00Z"
    }
  ]
}
```

---

### GET /analytics/kpis/definitions

List all available KPI definitions (read-only for non-admins).

**Response (200 OK):**
```json
{
  "items": [
    {
      "code": "sales_revenue",
      "name": "Sales Revenue",
      "description": "Total sales revenue",
      "unit": "USD",
      "formula": "SUM(sales.amount)",
      "frequency": "daily"
    }
  ],
  "total": 12
}
```

---

### POST /analytics/kpis/definitions

Create a new KPI definition.

**Authorization Required:** `system_admin` role

**Request Body:**
```json
{
  "code": "string (required, unique)",
  "name": "string (required)",
  "description": "string (optional)",
  "unit": "string (optional, e.g., 'USD', 'count')",
  "formula": "string (optional)",
  "frequency": "string (optional, 'daily', 'weekly', 'monthly')"
}
```

**Response (201 Created)**

---

### GET /analytics/trends

Retrieve trend data (time-series) for KPIs over a date range.

**Query Parameters:**
- `date_from` (required)
- `date_to` (required)
- `kpi_codes` (required, comma-separated)

**Response (200 OK):**
```json
{
  "trends": [
    {
      "code": "sales_revenue",
      "series": [
        { "date": "2026-04-01", "value": 148000 },
        { "date": "2026-04-02", "value": 150000 },
        { "date": "2026-04-03", "value": 149500 }
      ]
    }
  ]
}
```

---

## Report Endpoints

**Authorization Required:** `system_admin` or `operations_analyst` role + scope enforcement

### GET /reports/schedules

List all report schedules accessible to the user.

**Query Parameters:**
- `page`, `page_size` (optional)

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 1,
      "name": "Weekly Sales Report",
      "cron_expr": "0 8 * * 1",
      "timezone": "America/New_York",
      "output_format": "csv",
      "scope_json": "{\"city\": \"NYC\"}",
      "is_active": true,
      "created_at": "2026-01-01T00:00:00Z"
    }
  ],
  "total": 10
}
```

---

### POST /reports/schedules

Create a new report schedule.

**Request Body:**
```json
{
  "name": "string (required)",
  "cron_expr": "string (required, cron expression)",
  "timezone": "string (required, e.g., 'America/New_York')",
  "output_format": "string (required, 'csv', 'pdf', 'xlsx')",
  "scope_json": "string (optional, JSON scope filter)",
  "recipients": "string (optional, comma-separated emails)",
  "enabled": "boolean (optional, default: true)"
}
```

**Response (201 Created)**

---

### GET /reports/schedules/:id

Retrieve a specific report schedule.

**Response (200 OK)**

---

### PATCH /reports/schedules/:id

Update a report schedule.

**Request Body:**
```json
{
  "name": "string (optional)",
  "cron_expr": "string (optional)",
  "output_format": "string (optional)",
  "enabled": "boolean (optional)"
}
```

**Response (200 OK)**

---

### DELETE /reports/schedules/:id

Delete a report schedule (soft delete).

**Response (204 No Content)**

---

### POST /reports/schedules/:id/trigger

Manually trigger a report to execute immediately.

**Response (202 Accepted):**
```json
{
  "run_id": 42,
  "schedule_id": 1,
  "state": "running"
}
```

---

### GET /reports/runs

List all report run instances accessible to the user.

**Query Parameters:**
- `schedule_id` (optional) — Filter by schedule
- `state` (optional) — Filter by state ("running", "ready", "failed")
- `date_from` (optional, YYYY-MM-DD) — Filter by date range
- `date_to` (optional, YYYY-MM-DD)
- `page`, `page_size` (optional)

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 42,
      "schedule_id": 1,
      "state": "ready",
      "row_count": 1250,
      "started_at": "2026-04-14T08:00:00Z",
      "finished_at": "2026-04-14T08:05:30Z",
      "requested_by": 1,
      "format": "csv"
    }
  ],
  "total": 15
}
```

---

### GET /reports/runs/:id

Retrieve metadata for a specific report run.

**Response (200 OK)**

---

### GET /reports/runs/:id/download

Download the generated report file.

**Response (200 OK):** Binary file (CSV, PDF, or XLSX)

---

### GET /reports/runs/:id/access-check

Check if the current user has permission to access a specific report run.

**Response (200 OK):**
```json
{
  "has_access": true
}
```

**Response (200 OK, no access):**
```json
{
  "has_access": false
}
```

---

## Ingestion Endpoints

**Authorization Required:** `system_admin` or `operations_analyst` role

### GET /ingestion/sources

List all ingestion sources (data connectors).

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 1,
      "name": "Sales CRM",
      "type": "salesforce",
      "endpoint_url": "https://myorg.salesforce.com",
      "is_active": true,
      "last_sync_at": "2026-04-14T10:00:00Z"
    }
  ],
  "total": 5
}
```

---

### POST /ingestion/sources

Create a new ingestion source.

**Request Body:**
```json
{
  "name": "string (required)",
  "type": "string (required, e.g., 'salesforce', 'databricks', 'api')",
  "endpoint_url": "string (required)",
  "credentials_json": "string (optional, encrypted)"
}
```

**Response (201 Created)**

---

### GET /ingestion/jobs

List all ingestion job executions.

**Query Parameters:**
- `source_id` (optional)
- `state` (optional, 'pending', 'running', 'success', 'failed')
- `page`, `page_size` (optional)

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 1,
      "source_id": 1,
      "state": "success",
      "rows_processed": 5000,
      "rows_failed": 10,
      "started_at": "2026-04-14T09:00:00Z",
      "finished_at": "2026-04-14T09:15:30Z"
    }
  ],
  "total": 45
}
```

---

### POST /ingestion/jobs

Create and start a new ingestion job.

**Request Body:**
```json
{
  "source_id": 1,
  "target_entity": "string (optional, e.g., 'customer')"
}
```

**Response (201 Created)**

---

### GET /ingestion/jobs/:id

Retrieve a specific ingestion job.

**Response (200 OK)**

---

### POST /ingestion/jobs/:id/retry

Retry a failed ingestion job.

**Response (202 Accepted)**

---

### GET /ingestion/jobs/:id/checkpoints

List progress checkpoints for a long-running job.

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 1,
      "job_id": 1,
      "checkpoint_key": "batch_100",
      "rows_processed": 100,
      "last_seen_id": "abc123",
      "created_at": "2026-04-14T09:05:00Z"
    }
  ]
}
```

---

### GET /ingestion/jobs/:id/failures

List all failures and errors from a job.

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 1,
      "job_id": 1,
      "error_message": "Invalid email format",
      "row_data": {"name": "John", "email": "invalid"},
      "occurred_at": "2026-04-14T09:03:15Z"
    }
  ],
  "total": 10
}
```

---

## Audit Endpoints

**Authorization Required:** `system_admin` role only

### GET /audit/logs

List audit logs (immutable records of actions).

**Query Parameters:**
- `actor_user_id` (optional)
- `action_type` (optional, 'create', 'update', 'delete', 'export', etc.)
- `target_type` (optional)
- `date_from`, `date_to` (optional)
- `page`, `page_size` (optional)

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 1,
      "actor_user_id": 1,
      "action_type": "create",
      "target_type": "master_record",
      "target_id": "101",
      "details": "{\"entity_type\": \"sku\", \"natural_key\": \"SKU-001\"}",
      "ip_address": "192.168.1.100",
      "created_at": "2026-04-14T10:00:00Z"
    }
  ],
  "total": 5000
}
```

---

### GET /audit/logs/:id

Retrieve a specific audit log entry.

**Response (200 OK)**

---

### GET /audit/logs/search

Full-text search across audit logs.

**Query Parameters:**
- `q` (required) — Search query

**Response (200 OK):** (same format as GET /audit/logs)

---

### GET /audit/delete-requests

List pending delete requests (dual-control approval).

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 1,
      "entity_type": "audit_logs",
      "filter_json": "{\"date_before\": \"2025-01-01\"}",
      "reason": "Data retention policy cleanup",
      "status": "approved_once",
      "requested_by": 1,
      "approver1": 2,
      "approver2": null,
      "requested_at": "2026-04-13T10:00:00Z",
      "approved1_at": "2026-04-13T14:00:00Z"
    }
  ],
  "total": 3
}
```

---

### POST /audit/delete-requests

Create a new delete request.

**Request Body:**
```json
{
  "entity_type": "string (required, e.g., 'audit_logs', 'media')",
  "filter_json": "string (optional, JSON filter criteria)",
  "reason": "string (required)"
}
```

**Response (201 Created)**

---

### POST /audit/delete-requests/:id/approve

Approve a delete request.

**Request Body:**
```json
{
  "comment": "string (optional)"
}
```

**Response (200 OK)**
```json
{
  "status": "approved_once (or approved_twice once second approval is given)"
}
```

---

### POST /audit/delete-requests/:id/execute

Execute an approved delete request (requires both approvals).

**Response (202 Accepted)**
```json
{
  "message": "deletion process started",
  "rows_deleted": 1547
}
```

---

## Security Endpoints

**Authorization Required:** `system_admin` role only

### GET /security/sensitive-fields

List fields marked for encryption.

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 1,
      "entity": "user",
      "field_name": "email",
      "encryption_level": "pii",
      "enabled": true
    }
  ],
  "total": 12
}
```

---

### POST /security/sensitive-fields

Mark a field for encryption.

**Request Body:**
```json
{
  "entity": "string (required)",
  "field_name": "string (required)",
  "encryption_level": "string (required, 'pii', 'financial', 'health')"
}
```

**Response (201 Created)**

---

### GET /security/keys

List encryption keys and their rotations.

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 1,
      "key_version": 1,
      "created_at": "2026-01-01T00:00:00Z",
      "rotated_at": "2026-01-31T00:00:00Z",
      "next_rotation": "2026-02-28T00:00:00Z",
      "active": false
    },
    {
      "id": 2,
      "key_version": 2,
      "created_at": "2026-02-01T00:00:00Z",
      "active": true
    }
  ]
}
```

---

### POST /security/keys/rotate

Initiate encryption key rotation.

**Response (202 Accepted)**
```json
{
  "message": "key rotation started",
  "new_key_id": 3
}
```

---

### GET /security/retention-policies

List data retention policies.

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 1,
      "artifact_type": "audit_logs",
      "retention_days": 2555,
      "is_active": true,
      "description": "7 year retention for audit logs (compliance)"
    },
    {
      "id": 2,
      "artifact_type": "ingestion_failures",
      "retention_days": 90,
      "is_active": true,
      "description": "90 day retention for failed ingestion records"
    }
  ]
}
```

---

### POST /security/retention-policies

Create a new retention policy.

**Request Body:**
```json
{
  "artifact_type": "string (required)",
  "retention_days": 90,
  "description": "string (optional)"
}
```

**Response (201 Created)**

---

### POST /security/purge-runs/dry-run

Simulate data purge based on retention policies (no actual deletion).

**Request Body:**
```json
{
  "artifact_type": "string (required)"
}
```

**Response (200 OK):**
```json
{
  "artifact_type": "audit_logs",
  "would_delete": 5000,
  "affected_date_range": "before 2024-01-01"
}
```

---

### POST /security/purge-runs/execute

Execute data purge based on retention policies.

**Request Body:**
```json
{
  "artifact_type": "string (required)"
}
```

**Response (200 OK):**
```json
{
  "rows_deleted": 5000,
  "execution_time_ms": 12345
}
```

**Note:** Cannot purge `audit_logs` directly; requires explicit dual-control delete-request approval.

---

### GET /security/legal-holds

List active legal holds (blocks on data deletion).

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 1,
      "entity_type": "audit_logs",
      "filter_json": "{\"date_after\": \"2025-01-01\"}",
      "reason": "Litigation hold per legal request",
      "created_by": 1,
      "created_at": "2026-04-01T00:00:00Z",
      "released_at": null
    }
  ],
  "total": 2
}
```

---

### POST /security/legal-holds

Create a legal hold to prevent deletion.

**Request Body:**
```json
{
  "entity_type": "string (required)",
  "filter_json": "string (optional, JSON filter)",
  "reason": "string (required)"
}
```

**Response (201 Created)**

---

### POST /security/legal-holds/:id/release

Release a legal hold.

**Response (204 No Content)**

---

## Integration Endpoints

**Authorization Required:** `system_admin` role only

### GET /integrations/endpoints

List external integration endpoints.

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 1,
      "name": "Slack Notifications",
      "url": "https://hooks.slack.com/services/...",
      "type": "webhook",
      "enabled": true,
      "last_used_at": "2026-04-14T14:00:00Z"
    }
  ],
  "total": 8
}
```

---

### POST /integrations/endpoints

Create a new integration endpoint.

**Request Body:**
```json
{
  "name": "string (required)",
  "url": "string (required, https)",
  "type": "string (required, 'webhook', 'api', 'email')",
  "auth_type": "string (optional, 'none', 'bearer', 'api_key')",
  "auth_value": "string (optional, encrypted)"
}
```

**Response (201 Created)**

---

### POST /integrations/endpoints/:id/test

Test connectivity to an integration endpoint.

**Response (200 OK):**
```json
{
  "status": "success",
  "response_time_ms": 245,
  "message": "Connection successful"
}
```

---

### GET /integrations/deliveries

List message deliveries to external systems.

**Query Parameters:**
- `endpoint_id` (optional)
- `status` (optional, 'pending', 'success', 'failed', 'retrying')
- `date_from`, `date_to` (optional)

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 1,
      "endpoint_id": 1,
      "status": "success",
      "payload": "{\"event\": \"user.created\", \"user_id\": 42}",
      "http_status": 200,
      "response_time_ms": 145,
      "delivered_at": "2026-04-14T10:05:00Z"
    }
  ],
  "total": 542
}
```

---

### GET /integrations/connectors

List data connectors (sources/destinations for ETL).

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": 1,
      "name": "Salesforce",
      "type": "crm",
      "direction": "inbound",
      "enabled": true,
      "health_status": "healthy",
      "last_sync": "2026-04-14T10:00:00Z"
    }
  ],
  "total": 6
}
```

---

### POST /integrations/connectors

Create a new connector.

**Request Body:**
```json
{
  "name": "string (required)",
  "type": "string (required)",
  "direction": "string (required, 'inbound', 'outbound', 'bidirectional')",
  "config_json": "string (optional)"
}
```

**Response (201 Created)**

---

### POST /integrations/connectors/:id/health-check

Check connector health status.

**Response (200 OK):**
```json
{
  "status": "healthy",
  "message": "Connection active",
  "last_sync": "2026-04-14T10:00:00Z"
}
```

---

## Health Check

### GET /health

Public health check endpoint (no authentication required).

**Response (200 OK):**
```json
{
  "status": "healthy",
  "timestamp": "2026-04-14T14:30:00Z"
}
```

---

## Rate Limiting & Other Notes

- No explicit rate limiting is currently implemented; consider adding 100-1000 req/min per IP in production
- All datetime fields are returned in ISO 8601 format with UTC timezone
- Pagination defaults: page=1, page_size=50 (max 200)
- Large downloads (reports, exports) may take time; consider implementing async with download links
- WebSocket support not currently implemented; consider for real-time notifications
