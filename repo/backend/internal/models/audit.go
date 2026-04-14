package models

import (
	"time"
)

type AuditLog struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ActorUserID   *uint     `gorm:"index" json:"actor_user_id"`
	Actor         *User     `gorm:"foreignKey:ActorUserID" json:"-"`
	ActionType    string    `gorm:"size:100;not null;index" json:"action_type"`
	TargetType    string    `gorm:"size:100;not null;index" json:"target_type"`
	TargetID      string    `gorm:"size:255;not null;index" json:"target_id"`
	ScopeJSON     string    `gorm:"type:json" json:"scope_json"`
	SensitiveRead bool      `gorm:"default:false;index" json:"sensitive_read"`
	MetadataJSON  string    `gorm:"type:json" json:"metadata_json"`
	IPAddress     string    `gorm:"size:45" json:"ip_address"`
	UserAgent     string    `gorm:"size:512" json:"user_agent"`
	CreatedAt     time.Time `gorm:"index" json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

type AuditDeleteRequest struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	RequestedBy uint      `gorm:"not null;index" json:"requested_by"`
	Requester   User       `gorm:"foreignKey:RequestedBy" json:"-"`
	Reason      string     `gorm:"size:2000;not null" json:"reason"`
	State       string     `gorm:"size:50;not null;default:'pending';index" json:"state"`
	ApproverOne *uint      `json:"approver_one"`
	FirstApprover *User    `gorm:"foreignKey:ApproverOne" json:"-"`
	ApproverTwo *uint      `json:"approver_two"`
	SecondApprover *User   `gorm:"foreignKey:ApproverTwo" json:"-"`
	ExecutedAt  *time.Time `json:"executed_at"`
	TargetType  string     `gorm:"size:100;not null" json:"target_type"`
	TargetID    string     `gorm:"size:255;not null" json:"target_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (AuditDeleteRequest) TableName() string {
	return "audit_delete_requests"
}
