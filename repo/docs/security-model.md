# Security Model

This document describes the security architecture for the Multi-Org Data & Media Operations Hub.

## JWT Authentication

**Implementation**: `backend/internal/auth/auth_service.go`, `backend/internal/auth/auth_middleware.go`

| Parameter | Value | Source |
|---|---|---|
| Algorithm | HMAC-SHA256 | `jwt.SigningMethodHS256` |
| Access token duration | 30 minutes | `accessTokenDuration` constant |
| Refresh token duration | 12 hours | `refreshTokenDuration` constant |
| Idle timeout | 30 minutes | `idleTimeout` constant |
| Absolute session timeout | 12 hours | `absoluteTimeout` constant |
| Secret | Env `JWT_SECRET` | `config.go` |
| Issuer | Env `JWT_ISSUER` (default: `multi-org-hub`) | `config.go` |

### Token Claims

```go
type Claims struct {
    jwt.RegisteredClaims        // Subject (user ID), ID (JTI), Issuer, IssuedAt, ExpiresAt
    Role            string
    CityScope       string
    DepartmentScope string
}
```

### Session Lifecycle

1. Login creates a `Session` record with `jwt_jti`, `issued_at`, `last_activity_at`, `expires_at`.
2. Every authenticated request updates `last_activity_at`.
3. Idle timeout: `time.Since(session.LastActivityAt) > 30min` triggers revocation.
4. Absolute timeout: `time.Since(session.IssuedAt) > 12h` triggers revocation.
5. Logout sets `revoked_at` on the session.
6. Frontend idle tracking in `frontend/src/stores/auth.js` monitors activity events and auto-logs out after 30 minutes.

## Password Hashing (Argon2id)

**Implementation**: `backend/internal/auth/auth_service.go`

| Parameter | Value |
|---|---|
| Algorithm | Argon2id |
| Time (iterations) | 1 |
| Memory | 64 MB |
| Parallelism | 4 threads |
| Key length | 32 bytes |
| Salt length | 16 bytes (crypto/rand) |

Verification uses `crypto/subtle.ConstantTimeCompare`.

## Password Policy

**Implementation**: `ValidatePasswordComplexity()` in `auth_service.go`

- Minimum 12 characters
- At least 1 uppercase letter
- At least 1 lowercase letter
- At least 1 digit
- At least 1 symbol

## Account Lockout

| Parameter | Value |
|---|---|
| Max failed attempts | 5 |
| Lockout duration | 15 minutes |

Tracked via `failed_attempts` and `locked_until` in `users` table. Reset on successful login.

## AES-256-GCM Encryption

**Implementation**: `backend/internal/security/encryption.go`

| Parameter | Value |
|---|---|
| Algorithm | AES-256-GCM |
| Key size | 32 bytes (256 bits) |
| Nonce size | 12 bytes (standard GCM) |
| Feature flag | `ENABLE_BIOMETRIC` |

Output format: `nonce (12 bytes) || ciphertext || GCM tag`

Key wrapping via `WrapKey()`/`UnwrapKey()` for secure key storage.

## Key Rotation (Every 90 Days)

**Implementation**: `backend/internal/security/security_service.go` -- `RotateKey()`

1. Mark old key as `rotated` (atomic transaction).
2. Generate new 32-byte key via `crypto/rand`.
3. Store with `rotates_at` = now + 90 days.
4. Configurable via `KEY_ROTATION_DAYS` env.

## Sensitive Field Masking

| Pattern | Example |
|---|---|
| `last4` | `1234567890` -> `******7890` |
| `email` | `user@domain.com` -> `u***@domain.com` |
| `phone` | `(555) 123-4567` -> `(***) ***-4567` |
| `full` | `sensitive` -> `*********` |

`ShouldUnmask()` checks `unmask_roles_json` per field.

## Audit Log Immutability and Dual-Approval Deletion

- `LogAction()` is append-only (`db.Create()` only).
- Deletion requires `AuditDeleteRequest` with two distinct approvers (`approver_one`, `approver_two`).

## TLS Internal Trust Model

- `ENABLE_TLS` env (default: `false`)
- Docker bridge network for inter-service communication
- Nginx enforces security headers externally

## LAN-Only Egress Guard

**Implementation**: `backend/internal/middleware/middleware.go`

Allowed: `127.0.0.1`, `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`

Integration URLs validated via `ValidateEndpointURL()` in `integration_service.go`.

## Password Reset Workflow

1. Admin creates request (`POST /security/password-reset`)
2. Different admin approves (self-approval blocked)
3. Generates 32-byte one-time token (SHA-256 hashed), expires 1 hour
4. New password must pass complexity validation

## Security Headers (Nginx)

`X-Frame-Options: SAMEORIGIN`, `X-Content-Type-Options: nosniff`, `X-XSS-Protection: 1; mode=block`, `Referrer-Policy: strict-origin-when-cross-origin`

## Logging Redaction

Auto-redacted fields: `password`, `token`, `ssn`, `secret`, `authorization`, `cookie`

## Test Evidence

- `backend/tests/unit/auth_test.go`: Hashing, complexity, JWT, timeouts, lockout
- `backend/tests/unit/encryption_test.go`: AES-256-GCM round-trip, wrong key, tampering
- `backend/tests/unit/masking_test.go`: All masking patterns
- `backend/tests/api/auth_api_test.go`: Login, logout, refresh, locked accounts
- `frontend/src/tests/unit/auth.store.test.js`: Idle tracking, session persistence
