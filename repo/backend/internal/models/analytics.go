package models

import (
	"time"
)

type AnalyticsKPIDefinition struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Code           string    `gorm:"size:100;not null;uniqueIndex" json:"code"`
	DisplayName    string    `gorm:"size:255;not null" json:"display_name"`
	Description    string    `gorm:"size:1000" json:"description"`
	FormulaSQL     string    `gorm:"type:text;not null" json:"formula_sql"`
	DimensionsJSON string    `gorm:"type:json" json:"dimensions_json"`
	IsActive       bool      `gorm:"default:true;index" json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (AnalyticsKPIDefinition) TableName() string {
	return "analytics_kpi_definitions"
}

type ReportSchedule struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:255;not null" json:"name"`
	KPICode      string    `gorm:"size:100;index" json:"kpi_code"`
	CronExpr     string    `gorm:"size:100;not null" json:"cron_expr"`
	Timezone     string    `gorm:"size:100;not null;default:'America/New_York'" json:"timezone"`
	OutputFormat string    `gorm:"size:50;not null;default:'xlsx'" json:"output_format"`
	ScopeJSON    string    `gorm:"type:json" json:"scope_json"`
	Recipients   string    `gorm:"size:2000" json:"recipients"`
	IsActive     bool      `gorm:"default:true;index" json:"is_active"`
	CreatedBy    uint      `gorm:"not null" json:"created_by"`
	Creator      User      `gorm:"foreignKey:CreatedBy" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (ReportSchedule) TableName() string {
	return "report_schedules"
}

type ReportRun struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	ScheduleID     uint       `gorm:"not null;index" json:"schedule_id"`
	Schedule       ReportSchedule `gorm:"foreignKey:ScheduleID;constraint:OnDelete:CASCADE" json:"-"`
	State          string     `gorm:"size:50;not null;default:'pending';index" json:"state"`
	OutputPath     string     `gorm:"size:1000" json:"output_path"`
	FailureReason  string     `gorm:"size:2000" json:"failure_reason"`
	RequestedBy    *uint      `json:"requested_by"`
	Requester      *User      `gorm:"foreignKey:RequestedBy" json:"-"`
	RowCount       int        `gorm:"default:0" json:"row_count"`
	StartedAt      *time.Time `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (ReportRun) TableName() string {
	return "report_runs"
}
