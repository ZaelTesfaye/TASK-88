# Acceptance Evidence Map

Maps each reviewer scoring criterion to concrete evidence paths in the codebase.

## 1. Authentication and Session Security

| Criterion | Evidence | Path |
|---|---|---|
| JWT with Argon2id | Password hashing + token generation | `backend/internal/auth/auth_service.go` (lines 64-112, 151-195) |
| 30-min idle timeout | Idle check in middleware | `backend/internal/auth/auth_middleware.go` (line 73), `auth_service.go` (line 247) |
| 12-hour absolute timeout | Absolute check in middleware | `backend/internal/auth/auth_middleware.go` (line 80), `auth_service.go` (line 253) |
| Session revocation | Logout revokes session | `backend/internal/auth/auth_service.go` (lines 219-231) |
| Frontend idle tracking | 30-min timer with activity listeners | `frontend/src/stores/auth.js` (lines 7, 55-73) |
| Test: password hashing | Unit test | `backend/tests/unit/auth_test.go::TestHashAndVerifyPassword` |
| Test: password wrong | Unit test | `backend/tests/unit/auth_test.go::TestVerifyPasswordWrong` |
| Test: idle timeout | Unit test | `backend/tests/unit/auth_test.go::TestSessionIdleTimeout` |
| Test: absolute timeout | Unit test | `backend/tests/unit/auth_test.go::TestSessionAbsoluteTimeout` |
| Test: lockout | Unit test | `backend/tests/unit/auth_test.go::TestAccountLockout` |
| Test: login API | Integration test | `backend/tests/api/auth_api_test.go::TestLoginSuccess` |
| Test: locked login | Integration test | `backend/tests/api/auth_api_test.go::TestLoginLockedAccount` |
| Test: token refresh | Integration test | `backend/tests/api/auth_api_test.go::TestRefreshToken` |

## 2. Password Policy

| Criterion | Evidence | Path |
|---|---|---|
| Min 12, complexity | ValidatePasswordComplexity | `backend/internal/auth/auth_service.go` (lines 115-148) |
| Lockout after 5 | HandleFailedLogin | `backend/internal/auth/auth_service.go` (lines 257-276) |
| Test: valid passwords | Unit test | `backend/tests/unit/auth_test.go::TestPasswordComplexity` |
| Test: invalid passwords | Unit test | `backend/tests/unit/auth_test.go::TestPasswordComplexityMissing` |
| Test: lockout expiry | Unit test | `backend/tests/unit/auth_test.go::TestAccountLockoutExpiry` |

## 3. RBAC

| Criterion | Evidence | Path |
|---|---|---|
| 4-role permission matrix | permissionMatrix map | `backend/internal/rbac/rbac.go` (lines 19-64) |
| RequireRole middleware | Gin middleware | `backend/internal/rbac/rbac.go` (lines 76-96) |
| RequirePermission middleware | Gin middleware | `backend/internal/rbac/rbac.go` (lines 99-114) |
| Object-level scope | CheckObjectScope | `backend/internal/rbac/scope.go` (lines 11-30) |
| Query-level scope filter | EnforceScopeOnQuery | `backend/internal/rbac/scope.go` (lines 40-56) |
| Frontend route guards | beforeEach with role check | `frontend/src/router/index.js` (lines 68-86) |
| Test: all permissions | Unit test | `backend/tests/unit/rbac_test.go::TestSystemAdminHasAllPermissions` |
| Test: scope check | Unit test | `backend/tests/unit/rbac_test.go::TestScopeCheck` |
| Test: wildcard scope | Unit test | `backend/tests/unit/rbac_test.go::TestScopeWildcard` |
| Test: frontend guards | Frontend test | `frontend/src/tests/unit/router.guards.test.js` |

## 4. Encryption and Key Management

| Criterion | Evidence | Path |
|---|---|---|
| AES-256-GCM implementation | EncryptAES256GCM / DecryptAES256GCM | `backend/internal/security/encryption.go` |
| Key generation | GenerateKey (32 bytes, crypto/rand) | `backend/internal/security/encryption.go` (lines 78-84) |
| Key wrapping | WrapKey / UnwrapKey | `backend/internal/security/encryption.go` (lines 88-101) |
| Key rotation | RotateKey (90-day cycle) | `backend/internal/security/security_service.go` (lines 316-356) |
| Test: encrypt/decrypt | Unit test | `backend/tests/unit/encryption_test.go::TestAES256GCMEncryptDecrypt` |
| Test: wrong key | Unit test | `backend/tests/unit/encryption_test.go::TestAES256GCMWrongKey` |
| Test: key generation | Unit test | `backend/tests/unit/encryption_test.go::TestKeyGeneration` |

## 5. Sensitive Field Masking

| Criterion | Evidence | Path |
|---|---|---|
| Masking patterns | MaskValue with last4/email/phone/full | `backend/internal/security/security_service.go` (lines 137-178) |
| Role-based unmasking | ShouldUnmask | `backend/internal/security/security_service.go` (lines 181-204) |
| Test: last4 | Unit test | `backend/tests/unit/masking_test.go::TestMaskLast4` |
| Test: email | Unit test | `backend/tests/unit/masking_test.go::TestMaskEmail` |
| Test: phone | Unit test | `backend/tests/unit/masking_test.go::TestMaskPhone` |
| Test: full | Unit test | `backend/tests/unit/masking_test.go::TestMaskFull` |

## 6. Ingestion Pipeline

| Criterion | Evidence | Path |
|---|---|---|
| Job engine with priority queue | ProcessNextJob (priority DESC, FIFO) | `backend/internal/ingestion/job_engine.go` (lines 130-160) |
| Exponential backoff | backoffDurations: 1m, 5m, 10m | `backend/internal/ingestion/job_engine.go` (lines 35-39) |
| Checkpoint every 1000 | checkpointInterval constant | `backend/internal/ingestion/job_engine.go` (line 30) |
| Connector interface | Connector interface definition | `backend/internal/ingestion/connector.go` (lines 25-37) |
| Validation rules per entity | DefaultValidationRules | `backend/internal/ingestion/validation.go` (lines 39-68) |
| Test: backoff | Unit test | `backend/tests/unit/ingestion_test.go::TestExponentialBackoff` |
| Test: checkpoint | Unit test | `backend/tests/unit/ingestion_test.go::TestCheckpointWrite` |
| Test: state transitions | Unit test | `backend/tests/unit/ingestion_test.go::TestJobStateTransitions` |
| Test: SKU validation | Unit test | `backend/tests/unit/ingestion_test.go::TestValidationRuleSKU` |
| Test: CSV parsing | Unit test | `backend/tests/unit/ingestion_test.go::TestCSVParsing` |

## 7. Playback and LRC

| Criterion | Evidence | Path |
|---|---|---|
| LRC parser | ParseLRC with line+word level | `backend/internal/playback/lrc_parser.go` |
| Lyrics search | SearchLyrics (case-insensitive) | `backend/internal/playback/lrc_parser.go` (lines 117-132) |
| UTF-16 handling | ensureUTF8, decodeUTF16LE/BE | `backend/internal/playback/lrc_parser.go` (lines 183-248) |
| Supported formats | mp3, wav, flac, m4a, lrc | `backend/internal/playback/playback_service.go` (lines 13-16) |
| Test: basic LRC | Unit test | `backend/tests/unit/lrc_parser_test.go::TestParseLRCBasic` |
| Test: word level | Unit test | `backend/tests/unit/lrc_parser_test.go::TestParseLRCWordLevel` |
| Test: search | Unit test | `backend/tests/unit/lrc_parser_test.go::TestSearchLyrics` |
| Test: nearest line | Unit test | `backend/tests/unit/lrc_parser_test.go::TestFindNearestLine` |

## 8. Retention and Purge

| Criterion | Evidence | Path |
|---|---|---|
| Retention policies | RetentionPolicy model + CRUD | `backend/internal/models/retention.go`, `security_service.go` |
| Legal holds block purge | countPurgeEligible checks active holds | `backend/internal/security/security_service.go` (lines 557-589) |
| Dry-run purge | DryRunPurge (preview only) | `backend/internal/security/security_service.go` (lines 472-491) |
| Execute purge | ExecutePurge (records PurgeRun) | `backend/internal/security/security_service.go` (lines 494-531) |
| Test: dry-run | Unit test | `backend/tests/unit/retention_test.go::TestDryRunPurge` |
| Test: legal hold blocks | Unit test | `backend/tests/unit/retention_test.go::TestLegalHoldBlocksPurge` |
| Test: retention enforcement | Unit test | `backend/tests/unit/retention_test.go::TestRetentionPolicyEnforcement` |

## 9. Audit Trail

| Criterion | Evidence | Path |
|---|---|---|
| Append-only logging | LogAction (db.Create only) | `backend/internal/audit/audit_service.go` (lines 58-78) |
| Dual-approval deletion | AuditDeleteRequest with approver_one/two | `backend/internal/models/audit.go` (lines 27-45) |
| Audit action types | 15 action constants | `backend/internal/audit/audit_service.go` (lines 14-30) |

## 10. Error Contract

| Criterion | Evidence | Path |
|---|---|---|
| Unified error format | AppError struct | `backend/internal/errors/errors.go` (lines 10-16) |
| Correlation ID in errors | RespondWithError reads correlation_id | `backend/internal/errors/errors.go` (lines 57-63) |
| Status code mapping | ErrorHandlerMiddleware switch | `backend/internal/errors/errors.go` (lines 93-142) |
| Test: correlation ID | Integration test | `backend/tests/api/auth_api_test.go::TestLoginErrorIncludesCorrelationId` |

## 11. Infrastructure

| Criterion | Evidence | Path |
|---|---|---|
| Docker Compose | 3-service orchestration | `docker-compose.yml` |
| Health check | GET /health endpoint | `backend/internal/router/router.go` (lines 43-48) |
| Graceful shutdown | Signal handling + context timeout | `backend/cmd/server/main.go` (lines 66-80) |
| LAN egress guard | EgressGuardMiddleware | `backend/internal/middleware/middleware.go` (lines 32-45) |
| Security headers | Nginx config | `frontend/nginx.conf` (lines 60-63) |

## 12. Documentation

| Criterion | Evidence | Path |
|---|---|---|
| README | Project overview, quickstart | `README.md` |
| Architecture | System diagram, module boundaries | `docs/architecture.md` |
| API spec | All endpoints documented | `docs/api-spec.md` |
| RBAC matrix | Permission table | `docs/rbac-matrix.md` |
| Security model | Auth, encryption, masking | `docs/security-model.md` |
| Test coverage map | Test-to-requirement mapping | `docs/test-coverage-map.md` |
