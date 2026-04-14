# Status Code Policy

## Overview

All API responses follow a consistent error contract defined in `backend/internal/errors/errors.go`. Every error response includes a `correlationId` for request tracing.

## Standard Error Response Schema

```json
{
  "code": "ERROR_CODE",
  "message": "Human-readable description of the error",
  "details": { },
  "correlationId": "uuid-v4-string"
}
```

| Field | Type | Description |
|---|---|---|
| `code` | string | Machine-readable error code (see mapping below) |
| `message` | string | Human-readable error description |
| `details` | object/null | Additional context (optional, varies by error type) |
| `correlationId` | string | UUID correlating the request through logs and responses |

Source: `backend/internal/errors/errors.go:10-16`

## HTTP Status to Error Code Mapping

| HTTP Status | Error Code | Constructor | When Used |
|---|---|---|---|
| 200 | -- | -- | Successful operations (GET, POST create, PUT, DELETE) |
| 201 | -- | -- | Resource created (POST /master/:entity, POST /media) |
| 400 | `BAD_REQUEST` | `BadRequest(message, details)` | Malformed JSON, missing required fields, invalid parameters |
| 401 | `AUTH_REQUIRED` | `Unauthorized(message)` | Missing/invalid/expired token, revoked session, session not found |
| 403 | `FORBIDDEN` | `Forbidden(message)` | Insufficient role, account locked, out-of-scope, non-LAN IP |
| 404 | `NOT_FOUND` | `NotFound(message)` | Resource does not exist (record, version, job, media asset) |
| 409 | `CONFLICT` | `Conflict(message, details)` | Duplicate natural_key, version state conflict |
| 422 | `VALIDATION_ERROR` | `ValidationError(message, details)` | Business rule violation (password complexity, invalid entity type, missing columns) |
| 500 | `INTERNAL_ERROR` | `InternalError(message)` | Unhandled server errors, database failures, panic recovery |

Source: `backend/internal/errors/errors.go:29-55,93-142`

## Correlation ID

### Generation

The `RequestIDMiddleware` in `backend/internal/middleware/middleware.go:16-26` either:
1. Uses the `X-Correlation-ID` header from the incoming request (if provided)
2. Generates a new UUID v4 if no header is present

### Propagation

- Set in Gin context as `"correlation_id"`
- Returned in the `X-Correlation-ID` response header
- Included in every error response payload as `correlationId`
- Logged in structured log entries

### Frontend Generation

The Axios client (`frontend/src/api/client.js:13-15`) generates a correlation ID for every outgoing request:

```javascript
function generateCorrelationId() {
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}
```

This is attached to every request via the `X-Correlation-ID` header.

## Example Error Payloads

### 400 Bad Request

```json
{
  "code": "BAD_REQUEST",
  "message": "invalid request body",
  "details": "Key: 'loginRequest.Username' Error:Field validation for 'Username' failed on the 'required' tag",
  "correlationId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
}
```

### 401 Unauthorized

```json
{
  "code": "AUTH_REQUIRED",
  "message": "invalid or expired token",
  "correlationId": "f1e2d3c4-b5a6-7890-abcd-ef1234567890"
}
```

### 403 Forbidden (Role)

```json
{
  "code": "FORBIDDEN",
  "message": "insufficient role privileges",
  "correlationId": "11223344-5566-7788-99aa-bbccddeeff00"
}
```

### 403 Forbidden (Egress Guard)

```json
{
  "code": "FORBIDDEN",
  "message": "access denied: client IP 203.0.113.1 is not in allowed network range",
  "correlationId": "aabbccdd-eeff-0011-2233-445566778899"
}
```

### 403 Forbidden (Account Locked)

```json
{
  "code": "FORBIDDEN",
  "message": "account is temporarily locked due to too many failed login attempts",
  "correlationId": "12345678-abcd-ef01-2345-6789abcdef01"
}
```

### 404 Not Found

```json
{
  "code": "NOT_FOUND",
  "message": "master record 42 not found",
  "correlationId": "fedcba98-7654-3210-fedc-ba9876543210"
}
```

### 409 Conflict

```json
{
  "code": "CONFLICT",
  "message": "a sku record with key \"SKU001ABC\" already exists",
  "details": {
    "natural_key": "SKU001ABC"
  },
  "correlationId": "01234567-89ab-cdef-0123-456789abcdef"
}
```

### 422 Validation Error

```json
{
  "code": "VALIDATION_ERROR",
  "message": "SKU code must match pattern ^[A-Z0-9]{6,20}$ (6-20 uppercase alphanumeric characters)",
  "details": {
    "pattern": "^[A-Z0-9]{6,20}$",
    "value": "sku-invalid"
  },
  "correlationId": "abcdef01-2345-6789-abcd-ef0123456789"
}
```

### 422 Validation Error (Missing Columns)

```json
{
  "code": "VALIDATION_ERROR",
  "message": "missing required columns: code, name",
  "details": {
    "missing_columns": ["code", "name"],
    "expected_columns": ["code", "name", "description", "category"],
    "found_columns": ["description", "category", "price"]
  },
  "correlationId": "99887766-5544-3322-1100-ffeeddccbbaa"
}
```

### 500 Internal Error

```json
{
  "code": "INTERNAL_ERROR",
  "message": "An unexpected error occurred",
  "correlationId": "deadbeef-cafe-babe-dead-beefcafebabe"
}
```

## Panic Recovery

The `RecoveryMiddleware` in `backend/internal/middleware/middleware.go:74-92` catches panics and returns a 500 response with the correlation ID:

```go
c.AbortWithStatusJSON(http.StatusInternalServerError, &appErrors.AppError{
    Code:          "INTERNAL_ERROR",
    Message:       "An unexpected error occurred",
    CorrelationID: cid,
})
```

## Error Handler Middleware

The `ErrorHandlerMiddleware` in `backend/internal/errors/errors.go:93-142` processes any unhandled `gin.Error` values attached to the context. It maps `AppError.Code` to HTTP status codes using the same mapping table above.

## Frontend Error Handling

The Axios response interceptor (`frontend/src/api/client.js:29-61`) normalizes all error responses:

```javascript
const appError = {
  code: data?.code || `HTTP_${status}`,
  message: data?.message || error.message,
  details: data?.details || null,
  status,
  correlationId: error.response.headers?.['x-correlation-id'] || null,
};
```

For network errors (no response from server):
```javascript
{
  code: 'NETWORK_ERROR',
  message: 'Unable to reach the server. Check your connection.',
  status: 0,
}
```
