package audit

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"backend/internal/logging"
	"backend/internal/models"
)

// Action type constants for audit logging.
const (
	ActionCreate        = "CREATE"
	ActionUpdate        = "UPDATE"
	ActionDelete        = "DELETE"
	ActionRead          = "READ"
	ActionSensitiveRead = "SENSITIVE_READ"
	ActionLogin         = "LOGIN"
	ActionLogout        = "LOGOUT"
	ActionFailedLogin   = "FAILED_LOGIN"
	ActionExport        = "EXPORT"
	ActionImport        = "IMPORT"
	ActionActivate      = "ACTIVATE"
	ActionDeactivate    = "DEACTIVATE"
	ActionApprove       = "APPROVE"
	ActionReject        = "REJECT"
	ActionRevoke        = "REVOKE"
)

// AuditEntry holds the data needed to create a single audit log record.
type AuditEntry struct {
	ActorUserID   *uint
	ActionType    string
	TargetType    string
	TargetID      string
	ScopeJSON     string
	SensitiveRead bool
	MetadataJSON  string
	IPAddress     string
	UserAgent     string
}

// AuditFilter holds the parameters for querying audit logs.
type AuditFilter struct {
	ActionType string
	TargetType string
	ActorID    *uint
	StartDate  *time.Time
	EndDate    *time.Time
	Page       int
	PageSize   int
}

// LogAction creates a new audit log entry. This is append-only; no update
// or delete operations are ever performed on the audit_logs table.
func LogAction(db *gorm.DB, entry AuditEntry) error {
	record := models.AuditLog{
		ActorUserID:   entry.ActorUserID,
		ActionType:    entry.ActionType,
		TargetType:    entry.TargetType,
		TargetID:      entry.TargetID,
		ScopeJSON:     entry.ScopeJSON,
		SensitiveRead: entry.SensitiveRead,
		MetadataJSON:  entry.MetadataJSON,
		IPAddress:     entry.IPAddress,
		UserAgent:     entry.UserAgent,
	}

	if err := db.Create(&record).Error; err != nil {
		logging.Error("audit", "log_action",
			fmt.Sprintf("failed to create audit log entry: %v", err))
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// LogSensitiveRead is a convenience function to log a sensitive data read event.
func LogSensitiveRead(db *gorm.DB, userID uint, targetType, targetID string) error {
	return LogAction(db, AuditEntry{
		ActorUserID:   &userID,
		ActionType:    ActionSensitiveRead,
		TargetType:    targetType,
		TargetID:      targetID,
		SensitiveRead: true,
	})
}

// GetAuditLogs retrieves audit logs with filtering and pagination.
// Returns the matching logs, the total count, and any error.
func GetAuditLogs(db *gorm.DB, filter AuditFilter) ([]models.AuditLog, int64, error) {
	query := db.Model(&models.AuditLog{})

	if filter.ActionType != "" {
		query = query.Where("action_type = ?", filter.ActionType)
	}
	if filter.TargetType != "" {
		query = query.Where("target_type = ?", filter.TargetType)
	}
	if filter.ActorID != nil {
		query = query.Where("actor_user_id = ?", *filter.ActorID)
	}
	if filter.StartDate != nil {
		query = query.Where("created_at >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("created_at <= ?", *filter.EndDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 50
	}
	if filter.PageSize > 200 {
		filter.PageSize = 200
	}

	offset := (filter.Page - 1) * filter.PageSize

	var logs []models.AuditLog
	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(filter.PageSize).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve audit logs: %w", err)
	}

	return logs, total, nil
}
