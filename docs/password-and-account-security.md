# Password and Account Security

## Password Policy

**Implementation**: `backend/internal/auth/auth_service.go:115-148`

### Complexity Requirements

| Rule | Minimum | Enforcement | Code Reference |
|---|---|---|---|
| Length | 12 characters | `len(password) < minPasswordLength` | `auth_service.go:117-119` |
| Uppercase letter | At least 1 | `unicode.IsUpper(ch)` | `auth_service.go:123` |
| Lowercase letter | At least 1 | `unicode.IsLower(ch)` | `auth_service.go:125` |
| Digit | At least 1 | `unicode.IsDigit(ch)` | `auth_service.go:127` |
| Symbol/Punctuation | At least 1 | `unicode.IsPunct(ch) \|\| unicode.IsSymbol(ch)` | `auth_service.go:129` |

### Validation Function

```go
func ValidatePasswordComplexity(password string) error
```

Returns an error message specifying which rule failed. Called during password changes and resets.

### Test Coverage

| Test | File | Validates |
|---|---|---|
| `TestPasswordComplexity` | `backend/tests/unit/auth_test.go:76-89` | 4 valid passwords pass all rules |
| `TestPasswordComplexityMissing` | `backend/tests/unit/auth_test.go:91-111` | 5 invalid passwords (too short, no upper, no lower, no digit, no symbol) are rejected |

## Password Hashing

**Algorithm**: Argon2id (memory-hard, resistant to GPU/ASIC attacks)

**Parameters** (defined in `backend/internal/auth/auth_service.go:22-27`):

| Parameter | Value | Description |
|---|---|---|
| Time | 1 iteration | Number of passes over memory |
| Memory | 64 MB (64 * 1024 KB) | Memory cost |
| Threads | 4 | Degree of parallelism |
| Key length | 32 bytes | Output hash size |
| Salt length | 16 bytes | Random salt per password |

### Hash Format

```
$argon2id$v=19$m=65536,t=1,p=4${base64_salt}${base64_hash}
```

### Verification

Uses `crypto/subtle.ConstantTimeCompare` to prevent timing attacks:

```go
if subtle.ConstantTimeCompare(expectedHash, computedHash) == 1 {
    return true, nil
}
```

Source: `backend/internal/auth/auth_service.go:108`

## Account Lockout

**Implementation**: `backend/internal/auth/auth_service.go:257-298`

### Parameters

| Parameter | Value | Constant | Source |
|---|---|---|---|
| Max failed attempts before lockout | 5 | `maxFailedAttempts = 5` | `auth_service.go:36` |
| Lockout duration | 15 minutes | `lockoutDuration = 15 * time.Minute` | `auth_service.go:37` |

### Lockout Flow

```
User attempts login
    |
    +-- Password correct?
    |       YES -> HandleSuccessfulLogin()
    |              - Reset failed_attempts to 0
    |              - Clear locked_until
    |
    +-- Password wrong?
            +-- HandleFailedLogin()
                - Increment failed_attempts
                - If failed_attempts >= 5:
                    Set locked_until = now + 15 min
                    Log warning: "account locked for user X after N failed attempts"
```

### Lock Check

```go
func IsAccountLocked(user *models.User) bool {
    if user.LockedUntil == nil {
        return false
    }
    return time.Now().Before(*user.LockedUntil)
}
```

The lock expires automatically: once `time.Now()` passes `LockedUntil`, the user can attempt login again.

Source: `backend/internal/auth/auth_service.go:293-298`

### User Model Fields

| Field | Type | Purpose |
|---|---|---|
| `FailedAttempts` | INT | Counter incremented on each failed login |
| `LockedUntil` | *time.Time | Lock expiry timestamp (nil = not locked) |

Source: `backend/internal/models/user.go:15-16`

### Test Coverage

| Test | Validates |
|---|---|
| `TestAccountLockout` | User with 5 failures + future lock is locked; user with 2 failures + nil lock is not |
| `TestAccountLockoutExpiry` | User with expired lock (1 min ago) is NOT locked; user with active lock (14 min future) IS locked |

Source: `backend/tests/unit/auth_test.go:251-295`

## Password Reset Workflow

**Implementation**: `backend/internal/security/security_service.go:209-293`

### Dual-Authorization Flow

```
Step 1: Request
  POST /security/password-reset
  Body: { user_id, reason }
  Actor: system_admin (requester)
  -> Creates PasswordResetRequest with status "pending"
  -> Expires in 24 hours

Step 2: Approval
  POST /security/password-reset/:id/approve
  Actor: system_admin (approver)
  Constraint: approver != requester
  -> Generates 32-byte random token (crypto/rand)
  -> SHA-256 hashes token for storage
  -> Sets status to "approved"
  -> Token expires in 1 hour
  -> Returns plaintext token to approver

Step 3: Token Use
  The one-time token is used to reset the password.
  New password must pass ValidatePasswordComplexity().
```

### Security Properties

| Property | Implementation |
|---|---|
| Dual authorization | `req.RequestedBy == approverID` check returns error |
| Token generation | 32 bytes from `crypto/rand` |
| Token storage | SHA-256 hash only (plaintext never stored) |
| Token expiry | 1 hour from approval |
| Request expiry | 24 hours from creation |

### Password Reset Request Model

| Field | Type | Description |
|---|---|---|
| `user_id` | BIGINT | Target user for reset |
| `requested_by` | BIGINT | Admin who requested |
| `approved_by` | BIGINT (nullable) | Admin who approved |
| `token_hash` | VARCHAR(255) | SHA-256 of one-time token |
| `expires_at` | DATETIME | When the token expires |
| `used_at` | DATETIME (nullable) | When the token was consumed |
| `status` | VARCHAR(50) | pending, approved, used |

Source: `backend/internal/models/security.go:38-56`

## Login Security Features

### Audit Logging

Every login attempt is recorded in the audit log:

| Event | Action Type | Details |
|---|---|---|
| Successful login | `LOGIN` | Records session JTI, IP, user agent |
| Failed login (wrong password) | `FAILED_LOGIN` | Records user ID, IP, user agent |
| Failed login (unknown user) | `FAILED_LOGIN` | Records attempted username, IP, user agent |
| Logout | `LOGOUT` | Records session JTI |

Source: `backend/internal/handlers/auth_handler.go:79-85,116-123,167-174`

### Session Security

| Feature | Implementation |
|---|---|
| Unique session per token | `Session.JwtJTI` maps to token's `jti` claim |
| Idle timeout enforcement | Checked on every authenticated request |
| Absolute timeout enforcement | Checked on every authenticated request |
| Session revocation on logout | `RevokeSession()` sets `revoked_at` |
| Activity tracking | `last_activity_at` updated on every request |
| IP + User Agent recorded | Stored in session table |

Source: `backend/internal/auth/auth_middleware.go:24-110`
