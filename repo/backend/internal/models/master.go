package models

import (
	"time"
)

type MasterRecord struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	EntityType  string    `gorm:"size:100;not null;index:idx_entity_key" json:"entity_type"`
	NaturalKey  string    `gorm:"size:255;not null;index:idx_entity_key" json:"natural_key"`
	PayloadJSON string    `gorm:"type:json" json:"payload_json"`
	Status      string    `gorm:"size:50;not null;default:'active';index" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (MasterRecord) TableName() string {
	return "master_records"
}

type MasterVersion struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	EntityType  string     `gorm:"size:100;not null;index:idx_version_entity_scope" json:"entity_type"`
	ScopeKey    string     `gorm:"size:255;not null;index:idx_version_entity_scope" json:"scope_key"`
	VersionNo   int        `gorm:"not null;index:idx_version_entity_scope" json:"version_no"`
	State       string     `gorm:"size:50;not null;default:'draft';index" json:"state"`
	CreatedBy   uint       `gorm:"not null" json:"created_by"`
	Creator     User       `gorm:"foreignKey:CreatedBy" json:"-"`
	ReviewedBy  *uint      `json:"reviewed_by"`
	Reviewer    *User      `gorm:"foreignKey:ReviewedBy" json:"-"`
	ActivatedBy *uint      `json:"activated_by"`
	Activator   *User      `gorm:"foreignKey:ActivatedBy" json:"-"`
	ActivatedAt *time.Time `json:"activated_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (MasterVersion) TableName() string {
	return "master_versions"
}

type MasterVersionItem struct {
	ID             uint         `gorm:"primaryKey" json:"id"`
	VersionID      uint         `gorm:"not null;index:idx_version_record" json:"version_id"`
	Version        MasterVersion `gorm:"foreignKey:VersionID;constraint:OnDelete:CASCADE" json:"-"`
	MasterRecordID uint         `gorm:"not null;index:idx_version_record" json:"master_record_id"`
	MasterRecord   MasterRecord `gorm:"foreignKey:MasterRecordID;constraint:OnDelete:CASCADE" json:"-"`
	CreatedAt      time.Time    `json:"created_at"`
}

func (MasterVersionItem) TableName() string {
	return "master_version_items"
}

type DeactivationEvent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	RecordID    uint      `gorm:"not null;index" json:"record_id"`
	Record      MasterRecord `gorm:"foreignKey:RecordID;constraint:OnDelete:CASCADE" json:"-"`
	Reason      string    `gorm:"size:1000;not null" json:"reason"`
	ActorUserID uint      `gorm:"not null;index" json:"actor_user_id"`
	Actor       User      `gorm:"foreignKey:ActorUserID" json:"-"`
	CreatedAt   time.Time `json:"created_at"`
}

func (DeactivationEvent) TableName() string {
	return "deactivation_events"
}
