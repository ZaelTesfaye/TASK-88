# Duplicate Detection Rules

## Overview

Duplicate detection operates at two levels:
1. **Exact match**: Prevents records with identical `entity_type + natural_key` from being created
2. **Fuzzy match**: Normalized key comparison for near-duplicate detection

**Primary source**: `backend/internal/masterdata/master_service.go`

## Per-Entity Match Keys

| Entity Type | Natural Key Field | Validation Pattern | Index Strategy |
|---|---|---|---|
| `sku` | `natural_key` (code) | `^[A-Z0-9]{6,20}$` | Composite index: `idx_entity_key(entity_type, natural_key)` |
| `color` | `natural_key` (code) | None (free-form) | Composite index: `idx_entity_key(entity_type, natural_key)` |
| `size` | `natural_key` (code) | None (free-form) | Composite index: `idx_entity_key(entity_type, natural_key)` |
| `season` | `natural_key` (code) | `^(SS\|FW)[0-9]{4}$` | Composite index: `idx_entity_key(entity_type, natural_key)` |
| `brand` | `natural_key` (code) | None (free-form) | Composite index: `idx_entity_key(entity_type, natural_key)` |
| `supplier` | `natural_key` (code) | None (free-form) | Composite index: `idx_entity_key(entity_type, natural_key)` |
| `customer` | `natural_key` (code) | `^[A-Z0-9]{4,20}$` (via validation.go) | Composite index: `idx_entity_key(entity_type, natural_key)` |

Source: `backend/internal/masterdata/master_service.go:44-48`, `backend/internal/ingestion/validation.go:39-68`

## Normalization Rules

The `normalizeKey()` function applies the following transformations for fuzzy matching:

```go
func normalizeKey(key string) string {
    key = strings.TrimSpace(key)       // Remove leading/trailing whitespace
    key = strings.ToUpper(key)         // Convert to uppercase
    key = strings.ReplaceAll(key, " ", "")   // Remove spaces
    key = strings.ReplaceAll(key, "-", "")   // Remove hyphens
    key = strings.ReplaceAll(key, "_", "")   // Remove underscores
    key = strings.ReplaceAll(key, ".", "")   // Remove dots
    return key
}
```

Source: `backend/internal/masterdata/master_service.go:730-738`

### Normalization Examples

| Raw Input | Normalized | Match? |
|---|---|---|
| `SKU-001-ABC` | `SKU001ABC` | -- |
| `sku_001_abc` | `SKU001ABC` | Same as above |
| `SKU 001 ABC` | `SKU001ABC` | Same as above |
| `sku.001.abc` | `SKU001ABC` | Same as above |
| `SKU001ABC` | `SKU001ABC` | Same as above |

## Exact Duplicate Check (On Create)

**Implementation**: `backend/internal/masterdata/master_service.go:237-248`

When creating a new record, the system checks for an exact match on `entity_type + natural_key`:

```go
var existingCount int64
s.db.Model(&models.MasterRecord{}).
    Where("entity_type = ? AND natural_key = ?", entityType, naturalKey).
    Count(&existingCount)

if existingCount > 0 {
    return nil, appErrors.Conflict(
        fmt.Sprintf("a %s record with key %q already exists", entityType, naturalKey),
        map[string]interface{}{"natural_key": naturalKey},
    )
}
```

**Response on conflict (HTTP 409)**:
```json
{
  "code": "CONFLICT",
  "message": "a sku record with key \"SKU001ABC\" already exists",
  "details": {
    "natural_key": "SKU001ABC"
  },
  "correlationId": "abc-123-def"
}
```

## Fuzzy Duplicate Check

**Implementation**: `backend/internal/masterdata/master_service.go:360-389`

The `CheckDuplicates()` function searches for near-matches using normalized comparison:

```sql
WHERE entity_type = ?
  AND (
    UPPER(REPLACE(REPLACE(natural_key, ' ', ''), '-', '')) LIKE ?
    OR UPPER(natural_key) = ?
  )
LIMIT 20
```

This catches records where the raw key differs only by whitespace, hyphens, or case.

## Uniqueness Check on Update

**Implementation**: `backend/internal/masterdata/master_service.go:285-298`

When updating a record's natural key, the system checks for uniqueness excluding the record itself:

```go
s.db.Model(&models.MasterRecord{}).
    Where("entity_type = ? AND natural_key = ? AND id != ?", record.EntityType, newKey, id).
    Count(&count)
```

## Import-Time Duplicate Detection

**Implementation**: `backend/internal/masterdata/master_service.go:513-536`

During bulk CSV/XLSX import, each row is checked against existing records:

```go
s.db.Model(&models.MasterRecord{}).
    Where("entity_type = ? AND natural_key = ?", entityType, naturalKey).
    Count(&existingCount)

if existingCount > 0 {
    result.Errors = append(result.Errors, ImportError{
        Row:     rowNum,
        Column:  "code",
        Message: fmt.Sprintf("record with key %q already exists", naturalKey),
        Value:   naturalKey,
    })
}
```

## Conflict Payload Examples

### Create Conflict

**Request**: `POST /api/v1/master/sku`
```json
{ "natural_key": "SKU001ABC", "payload_json": "{}" }
```

**Response 409**:
```json
{
  "code": "CONFLICT",
  "message": "a sku record with key \"SKU001ABC\" already exists",
  "details": { "natural_key": "SKU001ABC" },
  "correlationId": "lx3k8m2n-a1b2"
}
```

### Update Conflict

**Request**: `PUT /api/v1/master/sku/42`
```json
{ "natural_key": "SKU002XYZ" }
```

**Response 409** (if SKU002XYZ already exists on another record):
```json
{
  "code": "CONFLICT",
  "message": "a sku record with key \"SKU002XYZ\" already exists",
  "details": { "natural_key": "SKU002XYZ" },
  "correlationId": "mn9p4q7r-c3d4"
}
```

### Import Row Conflict

**Response 200** (import completes with errors):
```json
{
  "total_rows": 100,
  "success_count": 95,
  "error_count": 5,
  "errors": [
    {
      "row": 12,
      "column": "code",
      "message": "record with key \"SKU001ABC\" already exists",
      "value": "SKU001ABC"
    }
  ]
}
```

## Index Strategy

**Model definition**: `backend/internal/models/master.go:8-10`

```go
type MasterRecord struct {
    EntityType string `gorm:"size:100;not null;index:idx_entity_key"`
    NaturalKey string `gorm:"size:255;not null;index:idx_entity_key"`
}
```

The composite index `idx_entity_key(entity_type, natural_key)` supports:
- Fast exact-match lookups for duplicate detection
- Efficient filtering by entity type
- Uniqueness enforcement at the database level when combined with application checks

**SQL schema** (`backend/migrations/init.sql:114-116`):
```sql
INDEX idx_master_type (record_type),
INDEX idx_master_external (external_id),
INDEX idx_master_status (status),
```

## Entity-Specific Validation Patterns

### SKU Code
- Pattern: `^[A-Z0-9]{6,20}$`
- Compiled regex: `backend/internal/masterdata/master_service.go:45`
- Example valid: `SKU001ABC`, `WIDGET2025`
- Example invalid: `sku-001` (lowercase, hyphen), `AB` (too short)

### Season Code
- Pattern: `^(SS|FW)[0-9]{4}$`
- Compiled regex: `backend/internal/masterdata/master_service.go:46`
- Example valid: `SS2025`, `FW2024`
- Example invalid: `Spring2025`, `SS25`

### Supplier/Customer Phone
- Pattern: `^\([0-9]{3}\) [0-9]{3}-[0-9]{4}$`
- Compiled regex: `backend/internal/masterdata/master_service.go:47`
- Example valid: `(555) 123-4567`
- Example invalid: `555-123-4567`, `5551234567`

### Customer Code (via ingestion validation)
- Pattern: `^[A-Z0-9]{4,20}$`
- Source: `backend/internal/ingestion/validation.go:63`
