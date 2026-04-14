package models

import (
	"time"
)

type Session struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         uint       `gorm:"not null;index" json:"user_id"`
	User           User       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	JwtJTI         string     `gorm:"size:255;uniqueIndex;not null" json:"jwt_jti"`
	IssuedAt       time.Time  `gorm:"not null" json:"issued_at"`
	LastActivityAt time.Time  `gorm:"not null" json:"last_activity_at"`
	ExpiresAt      time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt      *time.Time `json:"revoked_at"`
	IPAddress      string     `gorm:"size:45" json:"ip_address"`
	UserAgent      string     `gorm:"size:512" json:"user_agent"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (Session) TableName() string {
	return "sessions"
}
