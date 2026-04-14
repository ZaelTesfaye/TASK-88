# Test Coverage Map

This document maps high-risk requirements to their corresponding test files and test names.

## Test Infrastructure

| Stack | Runner | Config | Command |
|---|---|---|---|
| Backend unit | `go test` | -- | `cd backend && go test ./tests/unit/... -v` |
| Backend API | `go test` | -- | `cd backend && go test ./tests/api/... -v` |
| Frontend | Vitest | `frontend/vitest.config.js` | `cd frontend && npx vitest run` |
| All | Shell script | -- | `./run_tests.sh` |

## Backend Unit Tests

### Authentication (`backend/tests/unit/auth_test.go`)

| Test Name | Requirement | Risk |
|---|---|---|
| `TestHashAndVerifyPassword` | Argon2id hashing round-trip | High |
| `TestVerifyPasswordWrong` | Wrong password rejection | High |
| `TestPasswordComplexity` | Valid passwords accepted | High |
| `TestPasswordComplexityMissing` | Missing complexity rejected (5 sub-tests) | High |
| `TestGenerateTokenPair` | JWT access+refresh generation with claims | High |
| `TestValidateTokenExpired` | Expired token rejected | High |
| `TestSessionIdleTimeout` | 30-min idle detection | High |
| `TestSessionAbsoluteTimeout` | 12-hour absolute timeout | High |
| `TestAccountLockout` | Lock after 5 failures | High |
| `TestAccountLockoutExpiry` | Lockout expires after 15 min | High |

### RBAC (`backend/tests/unit/rbac_test.go`)

| Test Name | Requirement | Risk |
|---|---|---|
| `TestSystemAdminHasAllPermissions` | system_admin gets all 19 permissions | High |
| `TestDataStewardPermissions` | data_steward allowed/denied permissions | High |
| `TestOperationsAnalystPermissions` | operations_analyst allowed/denied permissions | High |
| `TestStandardUserPermissions` | standard_user limited to view/playback/reports | High |
| `TestScopeCheck` | City/department scope matching | High |
| `TestScopeWildcard` | Wildcard and empty scope handling | Medium |

### Encryption (`backend/tests/unit/encryption_test.go`)

| Test Name | Requirement | Risk |
|---|---|---|
| `TestAES256GCMEncryptDecrypt` | Round-trip with multiple plaintexts, nonce uniqueness | High |
| `TestAES256GCMWrongKey` | Decryption fails with wrong key, short key, tampered ciphertext | High |
| `TestKeyGeneration` | 32-byte key, non-zero, unique, usable | High |

### Ingestion (`backend/tests/unit/ingestion_test.go`)

| Test Name | Requirement | Risk |
|---|---|---|
| `TestExponentialBackoff` | Backoff durations: 1m, 5m, 10m | High |
| `TestCheckpointWrite` | Checkpoint every 1000 records | High |
| `TestRetryLimit` | Max 3 retries, then failed_awaiting_ack | High |
| `TestJobStateTransitions` | Valid state transition chains | High |
| `TestValidationRuleSKU` | SKU code pattern validation | Medium |
| `TestValidationRuleSeason` | Season code pattern validation | Medium |
| `TestValidationRulePhone` | Phone format validation | Medium |
| `TestCSVParsing` | CSV parse with headers and BOM | Medium |
| `TestImportFileSizeLimit` | 50MB file rejection | Medium |

### LRC Parser (`backend/tests/unit/lrc_parser_test.go`)

| Test Name | Requirement | Risk |
|---|---|---|
| `TestParseLRCBasic` | Timestamp parsing, text extraction, sorting | Medium |
| `TestParseLRCWordLevel` | Word-level timing with start/end ms | Medium |
| `TestParseLRCMultipleTimestamps` | Multi-timestamp lines (chorus) | Medium |
| `TestParseLRCMetadata` | Metadata tag skipping | Low |
| `TestSearchLyrics` | Case-insensitive substring search | Medium |
| `TestFindNearestLine` | Binary search for closest line | Medium |
| `TestInvalidLRC` | Graceful handling of non-LRC content | Low |
| `TestEmptyLRC` | Empty/whitespace content handling | Low |

### Masking (`backend/tests/unit/masking_test.go`)

| Test Name | Requirement | Risk |
|---|---|---|
| `TestMaskLast4` | Last-4 masking pattern | High |
| `TestMaskEmail` | Email masking pattern | High |
| `TestMaskPhone` | Phone masking pattern | High |
| `TestMaskFull` | Full masking pattern | High |
| `TestMaskDefaultPattern` | Default/unknown pattern fallback | Medium |

### Retention (`backend/tests/unit/retention_test.go`)

| Test Name | Requirement | Risk |
|---|---|---|
| `TestDryRunPurge` | Dry-run counts eligible, no data modification | High |
| `TestLegalHoldBlocksPurge` | Active legal hold blocks all eligible records | High |
| `TestRetentionPolicyEnforcement` | Various retention day counts (30, 90, 365) | High |

## Backend API (Integration) Tests

### Auth API (`backend/tests/api/auth_api_test.go`)

| Test Name | Requirement | Risk |
|---|---|---|
| `TestLoginSuccess` | Valid credentials return token + user | High |
| `TestLoginInvalidCredentials` | Wrong password, nonexistent user, empty password | High |
| `TestLoginLockedAccount` | Locked account returns 403 | High |
| `TestLogoutRevokesSession` | Logout returns success message | High |
| `TestRefreshToken` | Refresh returns new distinct tokens | High |
| `TestProtectedRouteWithoutToken` | No token returns 401 | High |
| `TestProtectedRouteExpiredToken` | Expired token returns 401 | High |
| `TestProtectedRouteValidToken` | Valid token grants access | High |
| `TestLoginErrorIncludesCorrelationId` | Error response includes correlationId | Medium |

### Master API (`backend/tests/api/master_api_test.go`)

Tests master data CRUD operations through the API layer.

### RBAC API (`backend/tests/api/rbac_api_test.go`)

Tests role enforcement on protected API routes.

## Frontend Unit Tests

### Auth Store (`frontend/src/tests/unit/auth.store.test.js`)

Tests login, logout, session persistence, idle tracking, role checks.

### Context Store (`frontend/src/tests/unit/context.store.test.js`)

Tests org context switching and persistence.

### Router Guards (`frontend/src/tests/unit/router.guards.test.js`)

Tests route-level role guards and redirect behavior.

### Components (`frontend/src/tests/unit/components.test.js`)

Tests common UI components (AppChip, AppButton, AppDialog, etc.).

### Master Data (`frontend/src/tests/unit/master-data.test.js`)

Tests master data page interactions.

### Ingestion (`frontend/src/tests/unit/ingestion.test.js`)

Tests ingestion page job management interactions.

### Playback (`frontend/src/tests/unit/playback.test.js`)

Tests playback page media and lyrics functionality.

### Reports (`frontend/src/tests/unit/reports.test.js`)

Tests reports page schedule and run interactions.

### Security (`frontend/src/tests/unit/security.test.js`)

Tests security admin page sensitive fields and key management.
