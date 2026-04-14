# Retention Policy

This document describes the data retention and purge system for the Multi-Org Data & Media Operations Hub.

---

## Default Retention Period

Raw ingestion artifacts and transient operational data are subject to automatic purging after **30 days** (configurable via `RETENTION_PURGE_DAYS` env var).

---

## Artifact Types

The following artifact types are managed by the retention system:

| Artifact type         | Description                                      | Default retention |
|-----------------------|--------------------------------------------------|-------------------|
| `audit_logs`          | Immutable audit trail entries                    | 30 days           |
| `report_runs`         | Generated report execution records               | 30 days           |
| `ingestion_failures`  | Row-level ingestion failure records              | 30 days           |
| `ingestion_jobs`      | Completed or failed ingestion job records        | 30 days           |

Each artifact type has its own retention policy record in the `retention_policies` table, allowing independent configuration.

---

## Retention Policy Management

### List Policies

```
GET /api/v1/security/retention-policies
```

### Create Policy

```
POST /api/v1/security/retention-policies
```

```json
{
  "artifact_type": "audit_logs",
  "retention_days": 30,
  "legal_hold_enabled": false,
  "description": "Purge audit logs after 30 days",
  "is_active": true
}
```

### Update Policy

```
PUT /api/v1/security/retention-policies/:id
```

```json
{
  "retention_days": 60,
  "description": "Extended to 60 days per compliance"
}
```

---

## Legal Hold Override

A **legal hold** freezes all purge operations for the duration of the hold, regardless of retention age.

### How It Works

1. A `system_admin` creates a legal hold with a scope and reason.
2. While any active legal hold exists, purge operations for affected artifact types will report eligible records as "blocked by legal hold" and purge zero records.
3. When the hold is released, normal purge processing resumes.

### API

```
POST /api/v1/security/legal-holds
```

```json
{
  "scope_json": "{\"entity_type\":\"audit_logs\"}",
  "reason": "Pending litigation Case #12345"
}
```

```
POST /api/v1/security/legal-holds/:id/release
```

---

## Dry-Run Preview

Before executing a purge, operators can preview the impact with a dry run.

```
POST /api/v1/security/purge-runs/dry-run
```

```json
{ "artifact_type": "audit_logs" }
```

**Response:**

```json
{
  "artifact_type": "audit_logs",
  "eligible_count": 1500,
  "blocked_by_legal_hold": 0,
  "would_purge": 1500
}
```

If a legal hold is active:

```json
{
  "artifact_type": "audit_logs",
  "eligible_count": 1500,
  "blocked_by_legal_hold": 1500,
  "would_purge": 0
}
```

---

## Purge Execution

```
POST /api/v1/security/purge-runs/execute
```

```json
{ "artifact_type": "ingestion_failures" }
```

The purge:

1. Looks up the active retention policy for the artifact type.
2. Calculates the cutoff date (`now - retention_days`).
3. Checks for active legal holds; if any exist, blocks purging and records the blocked count.
4. Deletes eligible records older than the cutoff.
5. Records the purge run with counts.

### Purge eligibility by artifact type

| Artifact type        | Eligible when                                           |
|----------------------|---------------------------------------------------------|
| `audit_logs`         | `created_at < cutoff`                                   |
| `report_runs`        | `created_at < cutoff` AND `state` in (`ready`, `failed`, `skipped`) |
| `ingestion_failures` | `created_at < cutoff`                                   |
| `ingestion_jobs`     | `created_at < cutoff` AND `state` in (`completed`, `failed`) |

---

## Purge Audit Trail

Every purge execution is recorded in the `purge_runs` table:

| Field                        | Description                        |
|------------------------------|------------------------------------|
| `artifact_type`              | Type of artifact purged            |
| `dry_run`                    | Whether this was a dry run         |
| `initiated_by`               | User ID of the operator            |
| `started_at`                 | When the purge began               |
| `completed_at`               | When the purge finished            |
| `purged_count`               | Number of records deleted          |
| `blocked_by_legal_hold_count`| Number blocked by legal hold       |

```
GET /api/v1/security/purge-runs
```

---

## Configuration

| Environment variable     | Default | Description                          |
|--------------------------|---------|--------------------------------------|
| `RETENTION_PURGE_DAYS`   | `30`    | Default retention period in days     |

Source: `backend/internal/security/security_service.go`
