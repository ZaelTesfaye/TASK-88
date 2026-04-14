# Connector Plugin Contract

This document defines the interface and lifecycle for ingestion connectors in the Multi-Org Data & Media Operations Hub.

---

## Connector Interface

All connectors must implement the `Connector` interface defined in `backend/internal/ingestion/connector.go`:

```go
type Connector interface {
    Type() string
    Capabilities() []string
    ValidateConfig(config map[string]interface{}) error
    HealthCheck() (*HealthResult, error)
    Pull(cursor string, batchSize int) (*PullResult, error)
    AcknowledgeCheckpoint(cursor string) error
}
```

---

## Method Specifications

### Type() string

Returns the connector type identifier. Must be one of the registered constants.

| Constant           | Value             |
|--------------------|-------------------|
| `ConnectorFolder`  | `"folder"`        |
| `ConnectorShare`   | `"network_share"` |
| `ConnectorDB`      | `"database"`      |

### Capabilities() []string

Returns a list of capabilities the connector supports. Used for feature negotiation.

| Connector       | Capabilities                                       |
|-----------------|----------------------------------------------------|
| `folder`        | `pull`, `incremental`, `backfill`, `csv`, `xlsx`   |
| `network_share` | `pull`, `incremental`, `backfill`, `csv`, `xlsx`, `network_share` |
| `database`      | `pull`, `incremental`, `backfill`, `database`, `sql` |

### ValidateConfig(config map[string]interface{}) error

Validates connector configuration before use. Must return `nil` if valid, or an error describing the problem.

**Folder/Share required keys:**

| Key            | Type   | Required | Description               |
|----------------|--------|----------|---------------------------|
| `path`         | string | Yes      | Directory path to read    |
| `file_pattern` | string | No       | Glob pattern (default `*.csv`) |

**Database required keys:**

| Key             | Type   | Required | Description                      |
|-----------------|--------|----------|----------------------------------|
| `host`          | string | Yes      | Database host                    |
| `db_name`       | string | Yes      | Database name                    |
| `user`          | string | Yes      | Database user                    |
| `port`          | float64| No       | Port (default `3306`)            |
| `table`         | string | One of   | Table name to query              |
| `query`         | string | One of   | Custom SQL query                 |
| `cursor_column` | string | No       | Column for cursor pagination (default `id`) |

### HealthCheck() (*HealthResult, error)

Tests connectivity and returns a health status.

```go
type HealthResult struct {
    Healthy   bool      `json:"healthy"`
    Message   string    `json:"message"`
    CheckedAt time.Time `json:"checked_at"`
}
```

- Folder/Share: verifies the directory exists and is accessible
- Database: validates configuration completeness

### Pull(cursor string, batchSize int) (*PullResult, error)

Fetches records starting from the given cursor position.

```go
type PullResult struct {
    Records    []map[string]interface{} `json:"records"`
    NextCursor string                   `json:"next_cursor"`
    HasMore    bool                     `json:"has_more"`
}
```

- `cursor`: opaque string from a previous pull (empty string for first pull)
- `batchSize`: maximum records to return (default `1000` if <= 0)
- Each record includes a `_source_file` key for folder-based connectors

### AcknowledgeCheckpoint(cursor string) error

Confirms that a checkpoint was saved successfully. The connector may use this to track progress for incremental pulls.

---

## Connector Factory

The `ConnectorFactory` creates connector instances from definitions or import sources:

```go
factory := ingestion.NewConnectorFactory()

// From a ConnectorDefinition
conn, err := factory.Create(definition, configMap)

// From an ImportSource
conn, err := factory.CreateFromSource(source, configMap)
```

---

## Validation Lifecycle

```
1. Configuration submitted via API
        |
2. ConnectorFactory.Create() -- instantiate connector
        |
3. connector.ValidateConfig(config) -- validate config shape
        |  (fail -> 422 Validation Error)
        |
4. connector.HealthCheck() -- test connectivity
        |  (fail -> unhealthy status recorded)
        |
5. connector.Pull("", batchSize) -- initial data pull
        |
6. connector.AcknowledgeCheckpoint(cursor) -- confirm progress
        |
7. Repeat Pull/Acknowledge until HasMore == false
```

---

## Supported File Formats (Folder/Share)

| Extension | Reader       | Notes                              |
|-----------|--------------|------------------------------------|
| `.csv`    | `encoding/csv` | Header row required, lazy quotes |
| `.xlsx`   | `excelize`   | First sheet only, header row required |
| Other     | Skipped      | Unsupported extensions are silently ignored |

---

## Adding a New Connector Type

1. Define a new struct implementing the `Connector` interface.
2. Add a constant for the type (e.g., `ConnectorFTP = "ftp"`).
3. Register it in `ConnectorFactory.Create()` and `ConnectorFactory.CreateFromSource()`.
4. Add unit tests in `backend/tests/unit/ingestion_test.go`.
