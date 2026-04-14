package models

import (
	"time"
)

type SensitiveFieldRegistry struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	FieldKey        string    `gorm:"size:255;not null;uniqueIndex" json:"field_key"`
	DisplayName     string    `gorm:"size:255" json:"display_name"`
	MaskPattern     string    `gorm:"size:255;not null" json:"mask_pattern"`
	UnmaskRolesJSON string    `gorm:"type:json" json:"unmask_roles_json"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (SensitiveFieldRegistry) TableName() string {
	return "sensitive_field_registry"
}

type KeyRing struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	KeyID      string     `gorm:"size:255;not null;uniqueIndex" json:"key_id"`
	KeyPurpose string     `gorm:"size:100;not null;index" json:"key_purpose"`
	WrappedKey string     `gorm:"size:4000;not null" json:"-"`
	Algorithm  string     `gorm:"size:50;not null;default:'AES-256-GCM'" json:"algorithm"`
	RotatesAt  time.Time  `gorm:"not null;index" json:"rotates_at"`
	Status     string     `gorm:"size:50;not null;default:'active';index" json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (KeyRing) TableName() string {
	return "key_rings"
}

type PasswordResetRequest struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"not null;index" json:"user_id"`
	User        User       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	RequestedBy uint      `gorm:"not null" json:"requested_by"`
	Requester   User       `gorm:"foreignKey:RequestedBy" json:"-"`
	ApprovedBy  *uint      `json:"approved_by"`
	Approver    *User      `gorm:"foreignKey:ApprovedBy" json:"-"`
	TokenHash   string     `gorm:"size:255" json:"-"`
	ExpiresAt   time.Time  `gorm:"not null;index" json:"expires_at"`
	UsedAt      *time.Time `json:"used_at"`
	Status      string     `gorm:"size:50;not null;default:'pending';index" json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (PasswordResetRequest) TableName() string {
	return "password_reset_requests"
}
