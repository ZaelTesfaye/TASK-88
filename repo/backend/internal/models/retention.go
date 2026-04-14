package models

import (
	"time"
)

type RetentionPolicy struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	ArtifactType     string    `gorm:"size:100;not null;uniqueIndex" json:"artifact_type"`
	RetentionDays    int       `gorm:"not null" json:"retention_days"`
	LegalHoldEnabled bool      `gorm:"default:false" json:"legal_hold_enabled"`
	Description      string    `gorm:"size:1000" json:"description"`
	IsActive         bool      `gorm:"default:true" json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (RetentionPolicy) TableName() string {
	return "retention_policies"
}

type LegalHold struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	ScopeJSON  string     `gorm:"type:json;not null" json:"scope_json"`
	Reason     string     `gorm:"size:2000;not null" json:"reason"`
	CreatedBy  uint       `gorm:"not null;index" json:"created_by"`
	Creator    User       `gorm:"foreignKey:CreatedBy" json:"-"`
	ReleasedAt *time.Time `json:"released_at"`
	ReleasedBy *uint      `json:"released_by"`
	Releaser   *User      `gorm:"foreignKey:ReleasedBy" json:"-"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (LegalHold) TableName() string {
	return "legal_holds"
}

type PurgeRun struct {
	ID                      uint       `gorm:"primaryKey" json:"id"`
	ArtifactType            string     `gorm:"size:100;not null;index" json:"artifact_type"`
	DryRun                  bool       `gorm:"default:true" json:"dry_run"`
	PurgedCount             int        `gorm:"default:0" json:"purged_count"`
	BlockedByLegalHoldCount int        `gorm:"default:0" json:"blocked_by_legal_hold_count"`
	StartedAt               *time.Time `json:"started_at"`
	CompletedAt             *time.Time `json:"completed_at"`
	InitiatedBy             *uint      `json:"initiated_by"`
	Initiator               *User      `gorm:"foreignKey:InitiatedBy" json:"-"`
	CreatedAt               time.Time  `json:"created_at"`
}

func (PurgeRun) TableName() string {
	return "purge_runs"
}
