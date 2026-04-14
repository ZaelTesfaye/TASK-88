# RBAC Permission Matrix

This document defines the role-based access control model for the Multi-Org Data & Media Operations Hub.

---

## Roles

| Role                 | Description                                          |
|----------------------|------------------------------------------------------|
| `system_admin`       | Full access to all features, org management, security |
| `data_steward`       | Master data CRUD, import, review, version drafting    |
| `operations_analyst` | Analytics, reports, ingestion monitoring, read-only data |
| `standard_user`      | Read-only access to master data, playback, reports    |

---

## Permission Matrix

| Permission            | system_admin | data_steward | operations_analyst | standard_user |
|-----------------------|:------------:|:------------:|:------------------:|:-------------:|
| `master_data_view`    | Y            | Y            | Y                  | Y             |
| `master_data_crud`    | Y            | Y            |                    |               |
| `master_data_import`  | Y            | Y            |                    |               |
| `master_data_review`  | Y            | Y            |                    |               |
| `version_draft`       | Y            | Y            |                    |               |
| `version_activate`    | Y            |              |                    |               |
| `analytics_view`      | Y            |              | Y                  |               |
| `analytics_kpi`       | Y            |              | Y                  |               |
| `reports_view`        | Y            |              | Y                  | Y             |
| `reports_manage`      | Y            |              | Y                  |               |
| `reports_download`    | Y            |              | Y                  |               |
| `ingestion_view`      | Y            |              | Y                  |               |
| `ingestion_manage`    | Y            |              |                    |               |
| `playback_view`       | Y            | Y            | Y                  | Y             |
| `audit_view`          | Y            |              |                    |               |
| `audit_manage`        | Y            |              |                    |               |
| `security_manage`     | Y            |              |                    |               |
| `org_manage`          | Y            |              |                    |               |
| `integration_manage`  | Y            |              |                    |               |

Source: `backend/internal/rbac/rbac.go:permissionMatrix`

---

## Route-Level Enforcement

Routes are protected by two middleware types:

### RequireRole

Checks that the authenticated user's role is in the allowed set.

```go
rbac.RequireRole(rbac.SystemAdmin, rbac.OperationsAnalyst)
```

### RequirePermission

Checks that the user's role grants a specific permission key.

```go
rbac.RequirePermission("master_data_crud")
```

---

## Route-to-Permission Mapping

| Route group              | Required role or permission                              |
|--------------------------|----------------------------------------------------------|
| `POST /auth/login`       | None (public)                                            |
| `/org/*`                 | `system_admin` role                                      |
| `/context/*`             | `system_admin` role                                      |
| `GET /master/:entity`    | `master_data_view` permission                            |
| `POST /master/:entity`   | `master_data_crud` permission                            |
| `PUT /master/:entity/:id`| `master_data_crud` permission                            |
| `GET /versions/*`        | Any authenticated user                                   |
| `POST /versions/*`       | `version_draft` permission                               |
| `POST /versions/.../activate` | `system_admin` role                                 |
| `/ingestion/*`           | `system_admin` or `operations_analyst` role               |
| `/media/*`               | Any authenticated user (`playback_view`)                 |
| `/analytics/*`           | `system_admin` or `operations_analyst` role               |
| `/reports/*`             | `system_admin` or `operations_analyst` role               |
| `/audit/*`               | `system_admin` role                                      |
| `/security/*`            | `system_admin` role                                      |
| `/integrations/*`        | `system_admin` role                                      |

---

## Object-Level Scope Enforcement

Beyond route-level RBAC, the system enforces object-level scoping based on the user's `city_scope` and `department_scope` fields.

- `system_admin` has unrestricted scope (bypasses all scope checks).
- Other roles can only access records within their assigned city and department.
- A wildcard value (`*` or empty) means unrestricted for that dimension.

```go
// Middleware usage
rbac.RequireScope(targetCity, targetDept)

// Query-level filtering
rbac.ResolveScopeFilter(user, db)
```

The scope filter is applied as SQL `WHERE` clauses on `city` and `department` columns, ensuring database-level isolation.

Source: `backend/internal/rbac/scope.go`

---

## Org-Tree Scope Traversal

For org-context-aware queries, the system resolves descendant node IDs using breadth-first traversal of the `org_nodes` tree. Only records scoped to the user's node and its descendants are returned.

```go
nodeIDs, err := rbac.GetDescendantNodeIDs(db, currentNodeID)
```
