package models

import (
	"time"
)

type IntegrationEndpoint struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Name             string    `gorm:"size:255;not null" json:"name"`
	EventType        string    `gorm:"size:100;not null;index" json:"event_type"`
	URL              string    `gorm:"size:2000;not null" json:"url"`
	SigningSecretRef  string    `gorm:"size:255" json:"-"`
	HTTPMethod       string    `gorm:"size:10;not null;default:'POST'" json:"http_method"`
	HeadersJSON      string    `gorm:"type:json" json:"headers_json"`
	Enabled          bool      `gorm:"default:true;index" json:"enabled"`
	MaxRetries       int       `gorm:"default:3" json:"max_retries"`
	TimeoutSeconds   int       `gorm:"default:30" json:"timeout_seconds"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (IntegrationEndpoint) TableName() string {
	return "integration_endpoints"
}

type IntegrationDelivery struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	EndpointID  uint       `gorm:"not null;index" json:"endpoint_id"`
	Endpoint    IntegrationEndpoint `gorm:"foreignKey:EndpointID;constraint:OnDelete:CASCADE" json:"-"`
	EventID     string     `gorm:"size:255;not null;index" json:"event_id"`
	State       string     `gorm:"size:50;not null;default:'pending';index" json:"state"`
	PayloadJSON string     `gorm:"type:text" json:"payload_json"`
	ResponseCode *int      `json:"response_code"`
	ResponseBody string    `gorm:"type:text" json:"response_body"`
	Retries     int        `gorm:"default:0" json:"retries"`
	NextRetryAt *time.Time `gorm:"index" json:"next_retry_at"`
	DedupeKey   string     `gorm:"size:255;index" json:"dedupe_key"`
	DeliveredAt *time.Time `json:"delivered_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (IntegrationDelivery) TableName() string {
	return "integration_deliveries"
}

type ConnectorDefinition struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	Name              string     `gorm:"size:255;not null" json:"name"`
	ConnectorType     string     `gorm:"size:100;not null;uniqueIndex" json:"connector_type"`
	CapabilitiesJSON  string     `gorm:"type:json" json:"capabilities_json"`
	ConfigSchemaJSON  string     `gorm:"type:json" json:"config_schema_json"`
	HealthStatus      string     `gorm:"size:50;default:'unknown';index" json:"health_status"`
	LastHealthCheckAt *time.Time `json:"last_health_check_at"`
	IsActive          bool       `gorm:"default:true" json:"is_active"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (ConnectorDefinition) TableName() string {
	return "connector_definitions"
}
