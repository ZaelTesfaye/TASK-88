package models

import (
	"time"
)

type OrgNode struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ParentID   *uint     `gorm:"index" json:"parent_id"`
	Parent     *OrgNode  `gorm:"foreignKey:ParentID" json:"-"`
	LevelCode  string    `gorm:"size:50;not null;index" json:"level_code"`
	LevelLabel string    `gorm:"size:100;not null" json:"level_label"`
	Name       string    `gorm:"size:255;not null" json:"name"`
	City       string    `gorm:"size:255;index" json:"city"`
	Department string    `gorm:"size:255;index" json:"department"`
	IsActive   bool      `gorm:"default:true;index" json:"is_active"`
	SortOrder  int       `gorm:"default:0" json:"sort_order"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (OrgNode) TableName() string {
	return "org_nodes"
}

type ContextAssignment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_user_org" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	OrgNodeID uint      `gorm:"not null;uniqueIndex:idx_user_org;index" json:"org_node_id"`
	OrgNode   OrgNode   `gorm:"foreignKey:OrgNodeID;constraint:OnDelete:CASCADE" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ContextAssignment) TableName() string {
	return "context_assignments"
}
