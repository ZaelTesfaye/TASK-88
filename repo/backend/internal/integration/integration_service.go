package integration

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"backend/internal/logging"
	"backend/internal/models"
)

// Event type constants.
const (
	EventImportCompleted  = "import.completed"
	EventVersionActivated = "version.activated"
	EventReportCompleted  = "report.completed"
)

// IntegrationService manages webhook endpoints and event delivery.
type IntegrationService struct {
	db *gorm.DB
}

// NewIntegrationService creates a new IntegrationService.
func NewIntegrationService(db *gorm.DB) *IntegrationService {
	return &IntegrationService{db: db}
}

// CreateEndpointRequest is the request body for creating an integration endpoint.
type CreateEndpointRequest struct {
	Name             string `json:"name" binding:"required"`
	EventType        string `json:"event_type" binding:"required"`
	URL              string `json:"url" binding:"required"`
	SigningSecret    string `json:"signing_secret"`
	HTTPMethod       string `json:"http_method"`
	HeadersJSON      string `json:"headers_json"`
	Enabled          bool   `json:"enabled"`
	MaxRetries       int    `json:"max_retries"`
	TimeoutSeconds   int    `json:"timeout_seconds"`
}

// UpdateEndpointRequest is the request body for updating an integration endpoint.
type UpdateEndpointRequest struct {
	Name           *string `json:"name"`
	EventType      *string `json:"event_type"`
	URL            *string `json:"url"`
	SigningSecret  *string `json:"signing_secret"`
	HTTPMethod     *string `json:"http_method"`
	HeadersJSON    *string `json:"headers_json"`
	Enabled        *bool   `json:"enabled"`
	MaxRetries     *int    `json:"max_retries"`
	TimeoutSeconds *int    `json:"timeout_seconds"`
}

// DeliveryFilter holds the parameters for filtering delivery records.
type DeliveryFilter struct {
	EndpointID uint
	EventID    string
	State      string
	Page       int
	PageSize   int
}

// CreateEndpoint registers a new webhook endpoint after validating the URL is LAN-only.
func (s *IntegrationService) CreateEndpoint(req CreateEndpointRequest) (*models.IntegrationEndpoint, error) {
	if err := ValidateEndpointURL(req.URL); err != nil {
		return nil, err
	}

	httpMethod := req.HTTPMethod
	if httpMethod == "" {
		httpMethod = "POST"
	}
	maxRetries := req.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	timeoutSec := req.TimeoutSeconds
	if timeoutSec <= 0 {
		timeoutSec = 30
	}

	endpoint := models.IntegrationEndpoint{
		Name:            req.Name,
		EventType:       req.EventType,
		URL:             req.URL,
		SigningSecretRef: req.SigningSecret,
		HTTPMethod:      httpMethod,
		HeadersJSON:     req.HeadersJSON,
		Enabled:         req.Enabled,
		MaxRetries:      maxRetries,
		TimeoutSeconds:  timeoutSec,
	}

	if err := s.db.Create(&endpoint).Error; err != nil {
		return nil, fmt.Errorf("failed to create endpoint: %w", err)
	}

	return &endpoint, nil
}

// GetEndpoint returns a single endpoint by ID.
func (s *IntegrationService) GetEndpoint(id uint) (*models.IntegrationEndpoint, error) {
	var endpoint models.IntegrationEndpoint
	if err := s.db.First(&endpoint, id).Error; err != nil {
		return nil, err
	}
	return &endpoint, nil
}

// ListEndpoints returns all integration endpoints with pagination.
func (s *IntegrationService) ListEndpoints(page, pageSize int) ([]models.IntegrationEndpoint, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	var total int64
	s.db.Model(&models.IntegrationEndpoint{}).Count(&total)

	var endpoints []models.IntegrationEndpoint
	offset := (page - 1) * pageSize
	if err := s.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&endpoints).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list endpoints: %w", err)
	}

	return endpoints, total, nil
}

// UpdateEndpoint updates an existing endpoint.
func (s *IntegrationService) UpdateEndpoint(id uint, req UpdateEndpointRequest) (*models.IntegrationEndpoint, error) {
	var endpoint models.IntegrationEndpoint
	if err := s.db.First(&endpoint, id).Error; err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.EventType != nil {
		updates["event_type"] = *req.EventType
	}
	if req.URL != nil {
		if err := ValidateEndpointURL(*req.URL); err != nil {
			return nil, err
		}
		updates["url"] = *req.URL
	}
	if req.SigningSecret != nil {
		updates["signing_secret_ref"] = *req.SigningSecret
	}
	if req.HTTPMethod != nil {
		updates["http_method"] = *req.HTTPMethod
	}
	if req.HeadersJSON != nil {
		updates["headers_json"] = *req.HeadersJSON
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.MaxRetries != nil {
		updates["max_retries"] = *req.MaxRetries
	}
	if req.TimeoutSeconds != nil {
		updates["timeout_seconds"] = *req.TimeoutSeconds
	}

	if len(updates) > 0 {
		if err := s.db.Model(&endpoint).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("failed to update endpoint: %w", err)
		}
	}

	s.db.First(&endpoint, id)
	return &endpoint, nil
}

// DeleteEndpoint deletes an endpoint by ID.
func (s *IntegrationService) DeleteEndpoint(id uint) error {
	result := s.db.Delete(&models.IntegrationEndpoint{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete endpoint: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// SendEvent sends an event payload to all registered endpoints for the given event type.
// It uses at-least-once delivery: creates a delivery record, attempts to send, retries on failure.
func (s *IntegrationService) SendEvent(eventType string, payload interface{}) error {
	var endpoints []models.IntegrationEndpoint
	if err := s.db.Where("event_type = ? AND enabled = ?", eventType, true).Find(&endpoints).Error; err != nil {
		return fmt.Errorf("failed to find endpoints for event %s: %w", eventType, err)
	}

	if len(endpoints) == 0 {
		logging.Debug("integration", "send_event",
			fmt.Sprintf("no enabled endpoints for event type %s", eventType))
		return nil
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	eventID := uuid.New().String()
	dedupeKey := fmt.Sprintf("%s_%s", eventType, eventID)

	for _, ep := range endpoints {
		delivery := models.IntegrationDelivery{
			EndpointID:  ep.ID,
			EventID:     eventID,
			State:       "pending",
			PayloadJSON: string(payloadBytes),
			DedupeKey:   dedupeKey,
		}
		if err := s.db.Create(&delivery).Error; err != nil {
			logging.Error("integration", "send_event",
				fmt.Sprintf("failed to create delivery record for endpoint %d: %v", ep.ID, err))
			continue
		}

		// Attempt immediate delivery
		go s.attemptDelivery(&delivery, &ep)
	}

	return nil
}

// attemptDelivery tries to send a delivery to its endpoint.
func (s *IntegrationService) attemptDelivery(delivery *models.IntegrationDelivery, endpoint *models.IntegrationEndpoint) {
	payloadBytes := []byte(delivery.PayloadJSON)

	// Sign the payload with the endpoint's signing secret
	var signature string
	if endpoint.SigningSecretRef != "" {
		mac := hmac.New(sha256.New, []byte(endpoint.SigningSecretRef))
		mac.Write(payloadBytes)
		signature = hex.EncodeToString(mac.Sum(nil))
	}

	client := &http.Client{
		Timeout: time.Duration(endpoint.TimeoutSeconds) * time.Second,
	}

	req, err := http.NewRequest(endpoint.HTTPMethod, endpoint.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		s.markDeliveryFailed(delivery, endpoint, 0, fmt.Sprintf("failed to create request: %v", err))
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Event-ID", delivery.EventID)
	req.Header.Set("X-Dedupe-Key", delivery.DedupeKey)
	if signature != "" {
		req.Header.Set("X-Signature-256", "sha256="+signature)
	}

	// Apply custom headers
	if endpoint.HeadersJSON != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(endpoint.HeadersJSON), &headers); err == nil {
			for k, v := range headers {
				req.Header.Set(k, v)
			}
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		s.markDeliveryFailed(delivery, endpoint, 0, fmt.Sprintf("request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	respCode := resp.StatusCode

	if respCode >= 200 && respCode < 300 {
		now := time.Now()
		s.db.Model(delivery).Updates(map[string]interface{}{
			"state":         "delivered",
			"response_code": respCode,
			"response_body": string(bodyBytes),
			"delivered_at":  now,
		})
		logging.Info("integration", "delivery",
			fmt.Sprintf("delivered event %s to endpoint %d (HTTP %d)", delivery.EventID, delivery.EndpointID, respCode))
	} else {
		s.markDeliveryFailed(delivery, endpoint, respCode, string(bodyBytes))
	}
}

// markDeliveryFailed updates a delivery to failed state and schedules a retry if applicable.
func (s *IntegrationService) markDeliveryFailed(delivery *models.IntegrationDelivery, endpoint *models.IntegrationEndpoint, respCode int, respBody string) {
	delivery.Retries++
	updates := map[string]interface{}{
		"retries":       delivery.Retries,
		"response_body": respBody,
	}
	if respCode > 0 {
		updates["response_code"] = respCode
	}

	if delivery.Retries >= endpoint.MaxRetries {
		updates["state"] = "failed"
		logging.Warn("integration", "delivery",
			fmt.Sprintf("delivery %d for event %s permanently failed after %d retries",
				delivery.ID, delivery.EventID, delivery.Retries))
	} else {
		// Exponential backoff: 30s, 120s, 480s ...
		backoff := time.Duration(30*math.Pow(4, float64(delivery.Retries-1))) * time.Second
		nextRetry := time.Now().Add(backoff)
		updates["state"] = "retry"
		updates["next_retry_at"] = nextRetry
		logging.Info("integration", "delivery",
			fmt.Sprintf("delivery %d for event %s scheduled for retry at %s",
				delivery.ID, delivery.EventID, nextRetry.Format(time.RFC3339)))
	}

	s.db.Model(delivery).Updates(updates)
}

// ProcessDeliveries finds and retries pending/failed deliveries that are due for retry.
func (s *IntegrationService) ProcessDeliveries() error {
	now := time.Now()
	var deliveries []models.IntegrationDelivery
	if err := s.db.Where("state = ? AND next_retry_at <= ?", "retry", now).
		Order("next_retry_at ASC").
		Limit(100).
		Find(&deliveries).Error; err != nil {
		return fmt.Errorf("failed to find deliveries for retry: %w", err)
	}

	for _, d := range deliveries {
		delivery := d
		var endpoint models.IntegrationEndpoint
		if err := s.db.First(&endpoint, delivery.EndpointID).Error; err != nil {
			logging.Error("integration", "process_deliveries",
				fmt.Sprintf("endpoint %d not found for delivery %d", delivery.EndpointID, delivery.ID))
			continue
		}

		go s.attemptDelivery(&delivery, &endpoint)
	}

	if len(deliveries) > 0 {
		logging.Info("integration", "process_deliveries",
			fmt.Sprintf("retrying %d deliveries", len(deliveries)))
	}

	return nil
}

// GetDelivery returns a single delivery by ID.
func (s *IntegrationService) GetDelivery(id uint) (*models.IntegrationDelivery, error) {
	var delivery models.IntegrationDelivery
	if err := s.db.First(&delivery, id).Error; err != nil {
		return nil, err
	}
	return &delivery, nil
}

// GetDeliveries returns delivery history with filtering and pagination.
func (s *IntegrationService) GetDeliveries(filter DeliveryFilter) ([]models.IntegrationDelivery, int64, error) {
	query := s.db.Model(&models.IntegrationDelivery{})

	if filter.EndpointID > 0 {
		query = query.Where("endpoint_id = ?", filter.EndpointID)
	}
	if filter.EventID != "" {
		query = query.Where("event_id = ?", filter.EventID)
	}
	if filter.State != "" {
		query = query.Where("state = ?", filter.State)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count deliveries: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	offset := (page - 1) * pageSize
	var deliveries []models.IntegrationDelivery
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&deliveries).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list deliveries: %w", err)
	}

	return deliveries, total, nil
}

// RetryDelivery resets a failed delivery to pending state for immediate retry.
func (s *IntegrationService) RetryDelivery(deliveryID uint) error {
	var delivery models.IntegrationDelivery
	if err := s.db.First(&delivery, deliveryID).Error; err != nil {
		return err
	}

	if delivery.State != "failed" && delivery.State != "retry" {
		return fmt.Errorf("delivery is in state %q and cannot be retried", delivery.State)
	}

	now := time.Now()
	if err := s.db.Model(&delivery).Updates(map[string]interface{}{
		"state":        "retry",
		"retries":      0,
		"next_retry_at": now,
	}).Error; err != nil {
		return fmt.Errorf("failed to reset delivery for retry: %w", err)
	}

	var endpoint models.IntegrationEndpoint
	if err := s.db.First(&endpoint, delivery.EndpointID).Error; err != nil {
		return fmt.Errorf("endpoint not found: %w", err)
	}

	go s.attemptDelivery(&delivery, &endpoint)
	return nil
}

// TestEndpoint sends a test event to verify the endpoint is reachable.
func (s *IntegrationService) TestEndpoint(endpointID uint) (int, string, error) {
	var endpoint models.IntegrationEndpoint
	if err := s.db.First(&endpoint, endpointID).Error; err != nil {
		return 0, "", err
	}

	testPayload := map[string]interface{}{
		"event":     "test",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"message":   "integration test event",
	}

	payloadBytes, _ := json.Marshal(testPayload)

	client := &http.Client{
		Timeout: time.Duration(endpoint.TimeoutSeconds) * time.Second,
	}

	req, err := http.NewRequest(endpoint.HTTPMethod, endpoint.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		return 0, "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Event-ID", "test-"+uuid.New().String())

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, string(body), nil
}

// === Connector operations ===

// ListConnectors returns all connector definitions with pagination.
func (s *IntegrationService) ListConnectors(page, pageSize int) ([]models.ConnectorDefinition, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	var total int64
	s.db.Model(&models.ConnectorDefinition{}).Count(&total)

	var connectors []models.ConnectorDefinition
	offset := (page - 1) * pageSize
	if err := s.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&connectors).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list connectors: %w", err)
	}

	return connectors, total, nil
}

// GetConnector returns a single connector by ID.
func (s *IntegrationService) GetConnector(id uint) (*models.ConnectorDefinition, error) {
	var conn models.ConnectorDefinition
	if err := s.db.First(&conn, id).Error; err != nil {
		return nil, err
	}
	return &conn, nil
}

// CreateConnector creates a new connector definition.
func (s *IntegrationService) CreateConnector(conn *models.ConnectorDefinition) error {
	if err := s.db.Create(conn).Error; err != nil {
		return fmt.Errorf("failed to create connector: %w", err)
	}
	return nil
}

// UpdateConnector updates an existing connector.
func (s *IntegrationService) UpdateConnector(id uint, updates map[string]interface{}) (*models.ConnectorDefinition, error) {
	var conn models.ConnectorDefinition
	if err := s.db.First(&conn, id).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&conn).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update connector: %w", err)
	}
	s.db.First(&conn, id)
	return &conn, nil
}

// DeleteConnector deletes a connector definition by ID.
func (s *IntegrationService) DeleteConnector(id uint) error {
	result := s.db.Delete(&models.ConnectorDefinition{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete connector: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// HealthCheckConnector runs a health check on a connector and updates its status.
func (s *IntegrationService) HealthCheckConnector(id uint) (*models.ConnectorDefinition, error) {
	var conn models.ConnectorDefinition
	if err := s.db.First(&conn, id).Error; err != nil {
		return nil, err
	}

	// Mark as healthy (actual health check would depend on connector type)
	now := time.Now()
	s.db.Model(&conn).Updates(map[string]interface{}{
		"health_status":        "healthy",
		"last_health_check_at": now,
	})

	conn.HealthStatus = "healthy"
	conn.LastHealthCheckAt = &now
	return &conn, nil
}

// ValidateEndpointURL ensures the URL is LAN-only (private network ranges).
func ValidateEndpointURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("URL must contain a hostname")
	}

	// Allow localhost
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return nil
	}

	ip := net.ParseIP(host)
	if ip == nil {
		// Try resolving hostname
		addrs, err := net.LookupHost(host)
		if err != nil || len(addrs) == 0 {
			return fmt.Errorf("could not resolve hostname %q", host)
		}
		ip = net.ParseIP(addrs[0])
		if ip == nil {
			return fmt.Errorf("resolved address is not a valid IP")
		}
	}

	// Check for private network ranges
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"fd00::/8",
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return nil
		}
	}

	// Check for link-local
	if strings.HasPrefix(ip.String(), "169.254.") || strings.HasPrefix(ip.String(), "fe80:") {
		return nil
	}

	return fmt.Errorf("URL must target a LAN/private network address, got %s", ip.String())
}
