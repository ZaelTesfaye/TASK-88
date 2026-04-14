package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"backend/internal/config"
	"backend/internal/logging"
	"backend/internal/models"
)

// SecurityService provides security administration operations.
type SecurityService struct {
	db *gorm.DB
}

// NewSecurityService creates a new SecurityService instance.
func NewSecurityService(db *gorm.DB) *SecurityService {
	return &SecurityService{db: db}
}

// === Sensitive Fields ===

// GetSensitiveFields returns all sensitive field registrations.
func (s *SecurityService) GetSensitiveFields() ([]models.SensitiveFieldRegistry, error) {
	var fields []models.SensitiveFieldRegistry
	if err := s.db.Where("is_active = ?", true).Order("field_key ASC").Find(&fields).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve sensitive fields: %w", err)
	}
	return fields, nil
}

// GetSensitiveFieldByID returns a single sensitive field by ID.
func (s *SecurityService) GetSensitiveFieldByID(id uint) (*models.SensitiveFieldRegistry, error) {
	var field models.SensitiveFieldRegistry
	if err := s.db.First(&field, id).Error; err != nil {
		return nil, err
	}
	return &field, nil
}

// CreateSensitiveField creates a new sensitive field registration.
func (s *SecurityService) CreateSensitiveField(field *models.SensitiveFieldRegistry) error {
	if err := s.db.Create(field).Error; err != nil {
		return fmt.Errorf("failed to create sensitive field: %w", err)
	}
	return nil
}

// UpdateFieldRequest is the request body for updating a sensitive field.
type UpdateFieldRequest struct {
	DisplayName     *string `json:"display_name"`
	MaskPattern     *string `json:"mask_pattern"`
	UnmaskRolesJSON *string `json:"unmask_roles_json"`
	IsActive        *bool   `json:"is_active"`
}

// UpdateSensitiveField updates a sensitive field registration by its field_key.
func (s *SecurityService) UpdateSensitiveField(fieldKey string, req UpdateFieldRequest) error {
	updates := make(map[string]interface{})
	if req.DisplayName != nil {
		updates["display_name"] = *req.DisplayName
	}
	if req.MaskPattern != nil {
		updates["mask_pattern"] = *req.MaskPattern
	}
	if req.UnmaskRolesJSON != nil {
		updates["unmask_roles_json"] = *req.UnmaskRolesJSON
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) == 0 {
		return nil
	}

	result := s.db.Model(&models.SensitiveFieldRegistry{}).Where("field_key = ?", fieldKey).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update sensitive field: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateSensitiveFieldByID updates a sensitive field registration by ID.
func (s *SecurityService) UpdateSensitiveFieldByID(id uint, req UpdateFieldRequest) error {
	updates := make(map[string]interface{})
	if req.DisplayName != nil {
		updates["display_name"] = *req.DisplayName
	}
	if req.MaskPattern != nil {
		updates["mask_pattern"] = *req.MaskPattern
	}
	if req.UnmaskRolesJSON != nil {
		updates["unmask_roles_json"] = *req.UnmaskRolesJSON
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) == 0 {
		return nil
	}

	result := s.db.Model(&models.SensitiveFieldRegistry{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update sensitive field: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// DeleteSensitiveField soft-deletes a sensitive field by ID.
func (s *SecurityService) DeleteSensitiveField(id uint) error {
	result := s.db.Model(&models.SensitiveFieldRegistry{}).Where("id = ?", id).Update("is_active", false)
	if result.Error != nil {
		return fmt.Errorf("failed to deactivate sensitive field: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// MaskValue applies masking to a field value based on the given pattern.
func (s *SecurityService) MaskValue(value string, pattern string) string {
	if value == "" {
		return value
	}

	switch pattern {
	case "last4":
		if len(value) <= 4 {
			return strings.Repeat("*", len(value))
		}
		return strings.Repeat("*", len(value)-4) + value[len(value)-4:]

	case "email":
		parts := strings.SplitN(value, "@", 2)
		if len(parts) != 2 {
			return strings.Repeat("*", len(value))
		}
		local := parts[0]
		if len(local) <= 1 {
			return "*@" + parts[1]
		}
		return string(local[0]) + strings.Repeat("*", len(local)-1) + "@" + parts[1]

	case "phone":
		digits := extractDigits(value)
		if len(digits) < 4 {
			return strings.Repeat("*", len(value))
		}
		last4 := digits[len(digits)-4:]
		return "(***) ***-" + last4

	case "full":
		return strings.Repeat("*", len(value))

	default:
		// Default: mask all but last 4
		if len(value) <= 4 {
			return strings.Repeat("*", len(value))
		}
		return strings.Repeat("*", len(value)-4) + value[len(value)-4:]
	}
}

// ShouldUnmask checks if a role can see unmasked values for a given field.
func (s *SecurityService) ShouldUnmask(fieldKey string, userRole string) bool {
	var field models.SensitiveFieldRegistry
	if err := s.db.Where("field_key = ? AND is_active = ?", fieldKey, true).First(&field).Error; err != nil {
		return false
	}

	if field.UnmaskRolesJSON == "" {
		return false
	}

	var roles []string
	if err := json.Unmarshal([]byte(field.UnmaskRolesJSON), &roles); err != nil {
		logging.Error("security", "should_unmask",
			fmt.Sprintf("failed to parse unmask_roles_json for field %s: %v", fieldKey, err))
		return false
	}

	for _, r := range roles {
		if r == userRole {
			return true
		}
	}
	return false
}

// === Password Reset ===

// CreatePasswordResetRequest creates a new password reset request.
func (s *SecurityService) CreatePasswordResetRequest(userID uint, requestedBy uint, reason string) (*models.PasswordResetRequest, error) {
	// Verify the target user exists
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("target user not found")
	}

	req := models.PasswordResetRequest{
		UserID:      userID,
		RequestedBy: requestedBy,
		Status:      "pending",
		ExpiresAt:   time.Now().Add(24 * time.Hour), // 24h for approval
	}

	if err := s.db.Create(&req).Error; err != nil {
		return nil, fmt.Errorf("failed to create password reset request: %w", err)
	}

	return &req, nil
}

// GetPasswordResetRequests returns all password reset requests with pagination.
func (s *SecurityService) GetPasswordResetRequests(page, pageSize int) ([]models.PasswordResetRequest, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	var total int64
	s.db.Model(&models.PasswordResetRequest{}).Count(&total)

	var requests []models.PasswordResetRequest
	offset := (page - 1) * pageSize
	if err := s.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&requests).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve password reset requests: %w", err)
	}

	return requests, total, nil
}

// ApprovePasswordResetRequest approves a request and returns a one-time token.
// The token is hashed before storage and expires in 1 hour.
func (s *SecurityService) ApprovePasswordResetRequest(requestID uint, approverID uint) (string, error) {
	var req models.PasswordResetRequest
	if err := s.db.First(&req, requestID).Error; err != nil {
		return "", fmt.Errorf("password reset request not found")
	}

	if req.Status != "pending" {
		return "", fmt.Errorf("request is not in pending state (current: %s)", req.Status)
	}

	if req.RequestedBy == approverID {
		return "", fmt.Errorf("the requester cannot approve their own request")
	}

	// Generate a one-time token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate reset token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// Hash the token for storage
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	expiresAt := time.Now().Add(1 * time.Hour)

	if err := s.db.Model(&req).Updates(map[string]interface{}{
		"approved_by": approverID,
		"token_hash":  tokenHash,
		"expires_at":  expiresAt,
		"status":      "approved",
	}).Error; err != nil {
		return "", fmt.Errorf("failed to approve password reset request: %w", err)
	}

	return token, nil
}

// === Key Rotation ===

// GetKeyRing returns all keys in the key ring.
func (s *SecurityService) GetKeyRing() ([]models.KeyRing, error) {
	var keys []models.KeyRing
	if err := s.db.Order("created_at DESC").Find(&keys).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve key ring: %w", err)
	}
	return keys, nil
}

// GetKeyByID returns a single key by ID.
func (s *SecurityService) GetKeyByID(id uint) (*models.KeyRing, error) {
	var key models.KeyRing
	if err := s.db.First(&key, id).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

// getMasterKey returns the 32-byte master key from configuration.
// The master key is provided as a hex-encoded string via MASTER_KEY_HEX.
func (s *SecurityService) getMasterKey() ([]byte, error) {
	cfg := config.GetConfig()
	if cfg.MasterKeyHex == "" {
		return nil, fmt.Errorf("MASTER_KEY_HEX is not configured; envelope encryption requires a master key")
	}
	masterKey, err := hex.DecodeString(cfg.MasterKeyHex)
	if err != nil {
		return nil, fmt.Errorf("MASTER_KEY_HEX is not valid hex: %w", err)
	}
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("MASTER_KEY_HEX must decode to exactly 32 bytes (got %d)", len(masterKey))
	}
	return masterKey, nil
}

// RotateKey creates a new key for the given purpose and marks the old one as "rotated".
// The new data-encryption key is envelope-encrypted with the master key before storage.
func (s *SecurityService) RotateKey(keyID string, purpose string) error {
	masterKey, err := s.getMasterKey()
	if err != nil {
		return fmt.Errorf("envelope encryption unavailable: %w", err)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Mark old key as rotated
		result := tx.Model(&models.KeyRing{}).
			Where("key_id = ? AND status = ?", keyID, "active").
			Updates(map[string]interface{}{
				"status": "rotated",
			})
		if result.Error != nil {
			return fmt.Errorf("failed to rotate old key: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("active key with id %q not found", keyID)
		}

		// Generate new data-encryption key material.
		newKeyBytes, err := GenerateKey()
		if err != nil {
			return fmt.Errorf("failed to generate new key: %w", err)
		}

		// Envelope-encrypt the data key with the master key.
		wrapped, err := WrapKey(newKeyBytes, masterKey)
		if err != nil {
			return fmt.Errorf("failed to envelope-encrypt new key: %w", err)
		}

		newKeyID := fmt.Sprintf("%s_%d", purpose, time.Now().UnixNano())

		newKey := models.KeyRing{
			KeyID:      newKeyID,
			KeyPurpose: purpose,
			WrappedKey: base64.StdEncoding.EncodeToString(wrapped),
			Algorithm:  "AES-256-GCM",
			RotatesAt:  time.Now().AddDate(0, 0, 90), // 90-day rotation
			Status:     "active",
		}

		if err := tx.Create(&newKey).Error; err != nil {
			return fmt.Errorf("failed to create new key: %w", err)
		}

		logging.Info("security", "rotate_key",
			fmt.Sprintf("rotated key %s, new key %s for purpose %s (envelope-encrypted)", keyID, newKeyID, purpose))
		return nil
	})
}

// UnwrapStoredKey decrypts a stored WrappedKey back to the raw data-encryption key.
func (s *SecurityService) UnwrapStoredKey(wrappedKeyB64 string) ([]byte, error) {
	masterKey, err := s.getMasterKey()
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(wrappedKeyB64)
	if err != nil {
		return nil, fmt.Errorf("failed to base64-decode wrapped key: %w", err)
	}
	return UnwrapKey(ciphertext, masterKey)
}

// === Retention & Purge ===

// GetRetentionPolicies returns all retention policies.
func (s *SecurityService) GetRetentionPolicies() ([]models.RetentionPolicy, error) {
	var policies []models.RetentionPolicy
	if err := s.db.Order("artifact_type ASC").Find(&policies).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve retention policies: %w", err)
	}
	return policies, nil
}

// UpdateRetentionRequest is the request body for updating a retention policy.
type UpdateRetentionRequest struct {
	RetentionDays    *int    `json:"retention_days"`
	LegalHoldEnabled *bool   `json:"legal_hold_enabled"`
	Description      *string `json:"description"`
	IsActive         *bool   `json:"is_active"`
}

// CreateRetentionPolicy creates a new retention policy.
func (s *SecurityService) CreateRetentionPolicy(policy *models.RetentionPolicy) error {
	if err := s.db.Create(policy).Error; err != nil {
		return fmt.Errorf("failed to create retention policy: %w", err)
	}
	return nil
}

// UpdateRetentionPolicy updates a retention policy.
func (s *SecurityService) UpdateRetentionPolicy(id uint, req UpdateRetentionRequest) error {
	updates := make(map[string]interface{})
	if req.RetentionDays != nil {
		updates["retention_days"] = *req.RetentionDays
	}
	if req.LegalHoldEnabled != nil {
		updates["legal_hold_enabled"] = *req.LegalHoldEnabled
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) == 0 {
		return nil
	}

	result := s.db.Model(&models.RetentionPolicy{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update retention policy: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// CreateLegalHoldRequest is the request body for creating a legal hold.
type CreateLegalHoldRequest struct {
	ScopeJSON string `json:"scope_json" binding:"required"`
	Reason    string `json:"reason" binding:"required"`
}

// CreateLegalHold creates a new legal hold.
func (s *SecurityService) CreateLegalHold(req CreateLegalHoldRequest, createdBy uint) (*models.LegalHold, error) {
	hold := models.LegalHold{
		ScopeJSON: req.ScopeJSON,
		Reason:    req.Reason,
		CreatedBy: createdBy,
	}

	if err := s.db.Create(&hold).Error; err != nil {
		return nil, fmt.Errorf("failed to create legal hold: %w", err)
	}

	return &hold, nil
}

// GetLegalHolds returns all legal holds (both active and released).
func (s *SecurityService) GetLegalHolds() ([]models.LegalHold, error) {
	var holds []models.LegalHold
	if err := s.db.Order("created_at DESC").Find(&holds).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve legal holds: %w", err)
	}
	return holds, nil
}

// ReleaseLegalHold releases a legal hold by setting released_at and released_by.
func (s *SecurityService) ReleaseLegalHold(id uint, releasedBy uint) error {
	var hold models.LegalHold
	if err := s.db.First(&hold, id).Error; err != nil {
		return err
	}

	if hold.ReleasedAt != nil {
		return fmt.Errorf("legal hold is already released")
	}

	now := time.Now()
	return s.db.Model(&hold).Updates(map[string]interface{}{
		"released_at": now,
		"released_by": releasedBy,
	}).Error
}

// PurgePreview describes the results of a dry-run purge.
type PurgePreview struct {
	ArtifactType       string `json:"artifact_type"`
	EligibleCount      int    `json:"eligible_count"`
	BlockedByLegalHold int    `json:"blocked_by_legal_hold"`
	WouldPurge         int    `json:"would_purge"`
}

// DryRunPurge previews how many artifacts would be purged for the given artifact type.
// audit_logs are excluded — deletion requires the dual-approval workflow.
func (s *SecurityService) DryRunPurge(artifactType string) (*PurgePreview, error) {
	if artifactType == "audit_logs" {
		return nil, fmt.Errorf("audit_logs cannot be purged via this endpoint; use the dual-approval audit delete-request workflow instead")
	}

	var policy models.RetentionPolicy
	if err := s.db.Where("artifact_type = ? AND is_active = ?", artifactType, true).First(&policy).Error; err != nil {
		return nil, fmt.Errorf("no active retention policy found for artifact type %q", artifactType)
	}

	cutoff := time.Now().AddDate(0, 0, -policy.RetentionDays)

	eligible, legalHoldBlocked, err := s.countPurgeEligible(artifactType, cutoff)
	if err != nil {
		return nil, err
	}

	return &PurgePreview{
		ArtifactType:       artifactType,
		EligibleCount:      eligible,
		BlockedByLegalHold: legalHoldBlocked,
		WouldPurge:         eligible - legalHoldBlocked,
	}, nil
}

// ExecutePurge purges artifacts past retention and not under legal hold, records the purge run.
// audit_logs are excluded from purge — deletion requires dual-approval via the audit delete-request workflow.
func (s *SecurityService) ExecutePurge(artifactType string, actorID uint) (*models.PurgeRun, error) {
	if artifactType == "audit_logs" {
		return nil, fmt.Errorf("audit_logs cannot be purged via this endpoint; use the dual-approval audit delete-request workflow instead")
	}

	var policy models.RetentionPolicy
	if err := s.db.Where("artifact_type = ? AND is_active = ?", artifactType, true).First(&policy).Error; err != nil {
		return nil, fmt.Errorf("no active retention policy found for artifact type %q", artifactType)
	}

	cutoff := time.Now().AddDate(0, 0, -policy.RetentionDays)
	now := time.Now()

	purgeRun := models.PurgeRun{
		ArtifactType: artifactType,
		DryRun:       false,
		InitiatedBy:  &actorID,
		StartedAt:    &now,
	}
	if err := s.db.Create(&purgeRun).Error; err != nil {
		return nil, fmt.Errorf("failed to create purge run: %w", err)
	}

	purgedCount, blockedCount, err := s.executePurgeForType(artifactType, cutoff)
	if err != nil {
		logging.Error("security", "execute_purge",
			fmt.Sprintf("purge failed for %s: %v", artifactType, err))
	}

	completedAt := time.Now()
	s.db.Model(&purgeRun).Updates(map[string]interface{}{
		"purged_count":               purgedCount,
		"blocked_by_legal_hold_count": blockedCount,
		"completed_at":               completedAt,
	})

	purgeRun.PurgedCount = purgedCount
	purgeRun.BlockedByLegalHoldCount = blockedCount
	purgeRun.CompletedAt = &completedAt

	return &purgeRun, nil
}

// GetPurgeRuns returns all purge runs.
func (s *SecurityService) GetPurgeRuns(page, pageSize int) ([]models.PurgeRun, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	var total int64
	s.db.Model(&models.PurgeRun{}).Count(&total)

	var runs []models.PurgeRun
	offset := (page - 1) * pageSize
	if err := s.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&runs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve purge runs: %w", err)
	}

	return runs, total, nil
}

// countPurgeEligible counts artifacts eligible for purge and those blocked by legal holds.
func (s *SecurityService) countPurgeEligible(artifactType string, cutoff time.Time) (eligible int, blocked int, err error) {
	// Count active legal holds
	var activeLegalHolds int64
	s.db.Model(&models.LegalHold{}).Where("released_at IS NULL").Count(&activeLegalHolds)

	switch artifactType {
	case "audit_logs":
		var count int64
		s.db.Model(&models.AuditLog{}).Where("created_at < ?", cutoff).Count(&count)
		eligible = int(count)
	case "report_runs":
		var count int64
		s.db.Model(&models.ReportRun{}).Where("created_at < ? AND state IN ?", cutoff, []string{"ready", "failed", "skipped"}).Count(&count)
		eligible = int(count)
	case "ingestion_failures":
		var count int64
		s.db.Model(&models.IngestionFailure{}).Where("created_at < ?", cutoff).Count(&count)
		eligible = int(count)
	case "ingestion_jobs":
		var count int64
		s.db.Model(&models.IngestionJob{}).Where("created_at < ? AND state IN ?", cutoff, []string{"completed", "failed"}).Count(&count)
		eligible = int(count)
	default:
		return 0, 0, fmt.Errorf("unsupported artifact type: %s", artifactType)
	}

	if activeLegalHolds > 0 {
		blocked = eligible
	}

	return eligible, blocked, nil
}

// executePurgeForType deletes records past retention that are not under legal hold.
func (s *SecurityService) executePurgeForType(artifactType string, cutoff time.Time) (purged int, blocked int, err error) {
	// Check for active legal holds
	var activeLegalHolds int64
	s.db.Model(&models.LegalHold{}).Where("released_at IS NULL").Count(&activeLegalHolds)

	if activeLegalHolds > 0 {
		// Count how many would be blocked
		eligible, legalBlocked, _ := s.countPurgeEligible(artifactType, cutoff)
		_ = eligible
		return 0, legalBlocked, nil
	}

	var result *gorm.DB
	switch artifactType {
	case "audit_logs":
		result = s.db.Where("created_at < ?", cutoff).Delete(&models.AuditLog{})
	case "report_runs":
		result = s.db.Where("created_at < ? AND state IN ?", cutoff, []string{"ready", "failed", "skipped"}).Delete(&models.ReportRun{})
	case "ingestion_failures":
		result = s.db.Where("created_at < ?", cutoff).Delete(&models.IngestionFailure{})
	case "ingestion_jobs":
		result = s.db.Where("created_at < ? AND state IN ?", cutoff, []string{"completed", "failed"}).Delete(&models.IngestionJob{})
	default:
		return 0, 0, fmt.Errorf("unsupported artifact type: %s", artifactType)
	}

	if result.Error != nil {
		return 0, 0, result.Error
	}

	return int(result.RowsAffected), 0, nil
}

// --- helpers ---

func extractDigits(s string) string {
	var digits strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	return digits.String()
}
