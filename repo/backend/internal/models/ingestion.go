package models

import (
	"time"
)

type ImportSource struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"size:255;not null;uniqueIndex" json:"name"`
	SourceType      string    `gorm:"size:100;not null;index" json:"source_type"`
	ConnectionJSON  string    `gorm:"type:json" json:"connection_json"`
	MappingRulesJSON string   `gorm:"type:json" json:"mapping_rules_json"`
	IsActive        bool      `gorm:"default:true;index" json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (ImportSource) TableName() string {
	return "import_sources"
}

type IngestionJob struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	ImportSourceID    uint       `gorm:"not null;index" json:"import_source_id"`
	ImportSource      ImportSource `gorm:"foreignKey:ImportSourceID;constraint:OnDelete:CASCADE" json:"-"`
	Priority          int        `gorm:"default:0;index" json:"priority"`
	State             string     `gorm:"size:50;not null;default:'pending';index" json:"state"`
	DependencyGroup   string     `gorm:"size:255;index" json:"dependency_group"`
	RetryCount        int        `gorm:"default:0" json:"retry_count"`
	MaxRetries        int        `gorm:"default:3" json:"max_retries"`
	NextRetryAt       *time.Time `gorm:"index" json:"next_retry_at"`
	TotalRecords      int        `gorm:"default:0" json:"total_records"`
	ProcessedRecords  int        `gorm:"default:0" json:"processed_records"`
	FailedRecords     int        `gorm:"default:0" json:"failed_records"`
	AcknowledgedBy    *uint      `json:"acknowledged_by"`
	Acknowledger      *User      `gorm:"foreignKey:AcknowledgedBy" json:"-"`
	AcknowledgedReason string    `gorm:"size:1000" json:"acknowledged_reason"`
	StartedAt         *time.Time `json:"started_at"`
	CompletedAt       *time.Time `json:"completed_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (IngestionJob) TableName() string {
	return "ingestion_jobs"
}

type IngestionCheckpoint struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	JobID            uint      `gorm:"not null;index" json:"job_id"`
	Job              IngestionJob `gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE" json:"-"`
	RecordsProcessed int       `gorm:"default:0" json:"records_processed"`
	CursorToken      string    `gorm:"size:1000" json:"cursor_token"`
	CreatedAt        time.Time `json:"created_at"`
}

func (IngestionCheckpoint) TableName() string {
	return "ingestion_checkpoints"
}

type IngestionFailure struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	JobID        uint      `gorm:"not null;index" json:"job_id"`
	Job          IngestionJob `gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE" json:"-"`
	RecordIndex  int       `gorm:"not null" json:"record_index"`
	RawData      string    `gorm:"type:text" json:"raw_data"`
	ErrorMessage string    `gorm:"size:2000;not null" json:"error_message"`
	ErrorCode    string    `gorm:"size:100;index" json:"error_code"`
	CreatedAt    time.Time `json:"created_at"`
}

func (IngestionFailure) TableName() string {
	return "ingestion_failures"
}
