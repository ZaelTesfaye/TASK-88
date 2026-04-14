package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/internal/audit"
	"backend/internal/auth"
	appErrors "backend/internal/errors"
	"backend/internal/ingestion"
	"backend/internal/logging"
	"backend/internal/models"
	"backend/internal/rbac"
)

// IngestionHandler handles ingestion-related HTTP requests.
type IngestionHandler struct {
	db      *gorm.DB
	engine  *ingestion.JobEngine
	factory *ingestion.ConnectorFactory
}

// NewIngestionHandler creates a new IngestionHandler.
func NewIngestionHandler(db *gorm.DB) *IngestionHandler {
	return &IngestionHandler{
		db:      db,
		engine:  ingestion.NewJobEngine(db),
		factory: ingestion.NewConnectorFactory(),
	}
}

// RegisterRoutes registers all ingestion routes.
func (h *IngestionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	ing := rg.Group("/ingestion")
	{
		ing.GET("/sources", h.ListSources)
		ing.POST("/sources", h.CreateSource)
		ing.GET("/sources/:id", h.GetSource)
		ing.PATCH("/sources/:id", h.UpdateSource)
		ing.PUT("/sources/:id", h.UpdateSource)
		ing.DELETE("/sources/:id", h.DeleteSource)

		ing.GET("/connectors/:id/health", h.ConnectorHealthCheck)
		ing.GET("/connectors/:id/capabilities", h.ConnectorCapabilities)

		ing.GET("/jobs", h.ListJobs)
		ing.POST("/jobs", h.CreateJob)
		ing.POST("/jobs/run", h.CreateJob)
		ing.GET("/jobs/:id", h.GetJob)
		ing.POST("/jobs/:id/retry", h.RetryJob)
		ing.POST("/jobs/:id/acknowledge", h.AcknowledgeJob)
		ing.GET("/jobs/:id/checkpoints", h.ListCheckpoints)
		ing.GET("/jobs/:id/failures", h.ListFailures)
	}
}

// ---------------------------------------------------------------------------
// Source endpoints
// ---------------------------------------------------------------------------

// createSourceRequest is the request body for creating an import source.
type createSourceRequest struct {
	Name           string                 `json:"name" binding:"required"`
	SourceType     string                 `json:"source_type" binding:"required"`
	ConnectionJSON map[string]interface{} `json:"connection_json" binding:"required"`
	MappingRules   map[string]interface{} `json:"mapping_rules_json"`
	IsActive       *bool                  `json:"is_active"`
}

// updateSourceRequest is the request body for updating an import source.
type updateSourceRequest struct {
	Name           *string                `json:"name"`
	SourceType     *string                `json:"source_type"`
	ConnectionJSON map[string]interface{} `json:"connection_json"`
	MappingRules   map[string]interface{} `json:"mapping_rules_json"`
	IsActive       *bool                  `json:"is_active"`
}

// ListSources returns all import sources, paginated.
func (h *IngestionHandler) ListSources(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	if user == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}
	if !rbac.HasPermission(user.Role, "ingestion_view") {
		appErrors.RespondForbidden(c, "insufficient permissions")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	query := h.db.Model(&models.ImportSource{})

	if sourceType := c.Query("source_type"); sourceType != "" {
		query = query.Where("source_type = ?", sourceType)
	}
	if isActive := c.Query("is_active"); isActive != "" {
		query = query.Where("is_active = ?", isActive == "true")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		appErrors.RespondInternalError(c, "failed to count sources")
		return
	}

	offset := (page - 1) * pageSize
	var sources []models.ImportSource
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&sources).Error; err != nil {
		appErrors.RespondInternalError(c, "failed to query sources")
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       sources,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

// CreateSource creates a new import source with config validation.
func (h *IngestionHandler) CreateSource(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	if user == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}
	if !rbac.HasPermission(user.Role, "ingestion_manage") {
		appErrors.RespondForbidden(c, "insufficient permissions")
		return
	}

	var req createSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondBadRequest(c, "invalid request body", err.Error())
		return
	}

	// Validate source type.
	validTypes := map[string]bool{
		ingestion.ConnectorFolder: true,
		ingestion.ConnectorShare:  true,
		ingestion.ConnectorDB:     true,
	}
	if !validTypes[req.SourceType] {
		appErrors.RespondBadRequest(c, "invalid source_type",
			fmt.Sprintf("must be one of: %s, %s, %s", ingestion.ConnectorFolder, ingestion.ConnectorShare, ingestion.ConnectorDB))
		return
	}

	// Validate connector config by attempting to create one.
	def := models.ConnectorDefinition{ConnectorType: req.SourceType}
	connector, err := h.factory.Create(def, req.ConnectionJSON)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid connection configuration", err.Error())
		return
	}
	if err := connector.ValidateConfig(req.ConnectionJSON); err != nil {
		appErrors.RespondValidationError(c, "connection configuration validation failed", err.Error())
		return
	}

	// Serialize JSON fields.
	connJSON, err := json.Marshal(req.ConnectionJSON)
	if err != nil {
		appErrors.RespondInternalError(c, "failed to serialize connection config")
		return
	}

	mappingJSON := "{}"
	if req.MappingRules != nil {
		b, err := json.Marshal(req.MappingRules)
		if err != nil {
			appErrors.RespondInternalError(c, "failed to serialize mapping rules")
			return
		}
		mappingJSON = string(b)
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	source := models.ImportSource{
		Name:             req.Name,
		SourceType:       req.SourceType,
		ConnectionJSON:   string(connJSON),
		MappingRulesJSON: mappingJSON,
		IsActive:         isActive,
	}

	if err := h.db.Create(&source).Error; err != nil {
		logging.Error("ingestion", "create_source", fmt.Sprintf("failed to create source: %v", err))
		appErrors.RespondInternalError(c, "failed to create import source")
		return
	}

	// Audit log.
	audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: &user.ID,
		ActionType:  audit.ActionCreate,
		TargetType:  "import_source",
		TargetID:    fmt.Sprintf("%d", source.ID),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
	})

	c.JSON(http.StatusCreated, source)
}

// GetSource returns a single import source.
func (h *IngestionHandler) GetSource(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	if user == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}
	if !rbac.HasPermission(user.Role, "ingestion_view") {
		appErrors.RespondForbidden(c, "insufficient permissions")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid source ID", nil)
		return
	}

	var source models.ImportSource
	if err := h.db.First(&source, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "import source not found")
			return
		}
		appErrors.RespondInternalError(c, "failed to get import source")
		return
	}

	c.JSON(http.StatusOK, source)
}

// UpdateSource updates an existing import source.
func (h *IngestionHandler) UpdateSource(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	if user == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}
	if !rbac.HasPermission(user.Role, "ingestion_manage") {
		appErrors.RespondForbidden(c, "insufficient permissions")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid source ID", nil)
		return
	}

	var source models.ImportSource
	if err := h.db.First(&source, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "import source not found")
			return
		}
		appErrors.RespondInternalError(c, "failed to get import source")
		return
	}

	var req updateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondBadRequest(c, "invalid request body", err.Error())
		return
	}

	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.SourceType != nil {
		validTypes := map[string]bool{
			ingestion.ConnectorFolder: true,
			ingestion.ConnectorShare:  true,
			ingestion.ConnectorDB:     true,
		}
		if !validTypes[*req.SourceType] {
			appErrors.RespondBadRequest(c, "invalid source_type", nil)
			return
		}
		updates["source_type"] = *req.SourceType
	}
	if req.ConnectionJSON != nil {
		// Validate the new connection config.
		sourceType := source.SourceType
		if req.SourceType != nil {
			sourceType = *req.SourceType
		}
		def := models.ConnectorDefinition{ConnectorType: sourceType}
		connector, err := h.factory.Create(def, req.ConnectionJSON)
		if err != nil {
			appErrors.RespondBadRequest(c, "invalid connection configuration", err.Error())
			return
		}
		if err := connector.ValidateConfig(req.ConnectionJSON); err != nil {
			appErrors.RespondValidationError(c, "connection configuration validation failed", err.Error())
			return
		}

		connJSON, err := json.Marshal(req.ConnectionJSON)
		if err != nil {
			appErrors.RespondInternalError(c, "failed to serialize connection config")
			return
		}
		updates["connection_json"] = string(connJSON)
	}
	if req.MappingRules != nil {
		b, err := json.Marshal(req.MappingRules)
		if err != nil {
			appErrors.RespondInternalError(c, "failed to serialize mapping rules")
			return
		}
		updates["mapping_rules_json"] = string(b)
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) == 0 {
		appErrors.RespondBadRequest(c, "no fields to update", nil)
		return
	}

	if err := h.db.Model(&source).Updates(updates).Error; err != nil {
		logging.Error("ingestion", "update_source", fmt.Sprintf("failed to update source: %v", err))
		appErrors.RespondInternalError(c, "failed to update import source")
		return
	}

	// Reload.
	h.db.First(&source, id)

	audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: &user.ID,
		ActionType:  audit.ActionUpdate,
		TargetType:  "import_source",
		TargetID:    fmt.Sprintf("%d", source.ID),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
	})

	c.JSON(http.StatusOK, source)
}

// DeleteSource deletes an import source (soft-delete by deactivating).
func (h *IngestionHandler) DeleteSource(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	if user == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}
	if !rbac.HasPermission(user.Role, "ingestion_manage") {
		appErrors.RespondForbidden(c, "insufficient permissions")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid source ID", nil)
		return
	}

	result := h.db.Model(&models.ImportSource{}).
		Where("id = ?", id).
		Update("is_active", false)
	if result.Error != nil {
		appErrors.RespondInternalError(c, "failed to delete import source")
		return
	}
	if result.RowsAffected == 0 {
		appErrors.RespondNotFound(c, "import source not found")
		return
	}

	audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: &user.ID,
		ActionType:  audit.ActionDeactivate,
		TargetType:  "import_source",
		TargetID:    fmt.Sprintf("%d", id),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
	})

	c.JSON(http.StatusOK, gin.H{"message": "import source deactivated"})
}

// ---------------------------------------------------------------------------
// Connector endpoints
// ---------------------------------------------------------------------------

// ConnectorHealthCheck performs a health check on a connector.
func (h *IngestionHandler) ConnectorHealthCheck(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	if user == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}
	if !rbac.HasPermission(user.Role, "ingestion_view") {
		appErrors.RespondForbidden(c, "insufficient permissions")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid connector ID", nil)
		return
	}

	// Look up as either a ConnectorDefinition or an ImportSource.
	var connDef models.ConnectorDefinition
	var connConfig map[string]interface{}
	var connType string

	if err := h.db.First(&connDef, id).Error; err == nil {
		connType = connDef.ConnectorType
		if err := json.Unmarshal([]byte(connDef.ConfigSchemaJSON), &connConfig); err != nil {
			connConfig = make(map[string]interface{})
		}
	} else {
		// Try ImportSource.
		var source models.ImportSource
		if err := h.db.First(&source, id).Error; err != nil {
			appErrors.RespondNotFound(c, "connector or source not found")
			return
		}
		connType = source.SourceType
		if err := json.Unmarshal([]byte(source.ConnectionJSON), &connConfig); err != nil {
			appErrors.RespondInternalError(c, "failed to parse source connection config")
			return
		}
	}

	def := models.ConnectorDefinition{ConnectorType: connType}
	connector, err := h.factory.Create(def, connConfig)
	if err != nil {
		appErrors.RespondInternalError(c, fmt.Sprintf("failed to create connector: %v", err))
		return
	}

	result, err := connector.HealthCheck()
	if err != nil {
		appErrors.RespondInternalError(c, fmt.Sprintf("health check failed: %v", err))
		return
	}

	c.JSON(http.StatusOK, result)
}

// ConnectorCapabilities returns the capabilities of a connector.
func (h *IngestionHandler) ConnectorCapabilities(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	if user == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}
	if !rbac.HasPermission(user.Role, "ingestion_view") {
		appErrors.RespondForbidden(c, "insufficient permissions")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid connector ID", nil)
		return
	}

	var connDef models.ConnectorDefinition
	var connConfig map[string]interface{}
	var connType string

	if err := h.db.First(&connDef, id).Error; err == nil {
		connType = connDef.ConnectorType
		if err := json.Unmarshal([]byte(connDef.ConfigSchemaJSON), &connConfig); err != nil {
			connConfig = make(map[string]interface{})
		}
	} else {
		var source models.ImportSource
		if err := h.db.First(&source, id).Error; err != nil {
			appErrors.RespondNotFound(c, "connector or source not found")
			return
		}
		connType = source.SourceType
		if err := json.Unmarshal([]byte(source.ConnectionJSON), &connConfig); err != nil {
			appErrors.RespondInternalError(c, "failed to parse source connection config")
			return
		}
	}

	def := models.ConnectorDefinition{ConnectorType: connType}
	connector, err := h.factory.Create(def, connConfig)
	if err != nil {
		appErrors.RespondInternalError(c, fmt.Sprintf("failed to create connector: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connector_type": connector.Type(),
		"capabilities":   connector.Capabilities(),
	})
}

// ---------------------------------------------------------------------------
// Job endpoints
// ---------------------------------------------------------------------------

// createJobRequest is the request body for creating an ingestion job.
type createJobRequest struct {
	ImportSourceID  uint   `json:"import_source_id" binding:"required"`
	Priority        int    `json:"priority"`
	DependencyGroup string `json:"dependency_group"`
	Mode            string `json:"mode"`
}

// acknowledgeJobRequest is the request body for acknowledging a failed job.
type acknowledgeJobRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// ListJobs returns the job queue with filters.
func (h *IngestionHandler) ListJobs(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	if user == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}
	if !rbac.HasPermission(user.Role, "ingestion_view") {
		appErrors.RespondForbidden(c, "insufficient permissions")
		return
	}

	var filter ingestion.JobFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		appErrors.RespondBadRequest(c, "invalid query parameters", err.Error())
		return
	}

	result, err := h.engine.GetJobQueue(filter)
	if err != nil {
		appErrors.RespondInternalError(c, "failed to list jobs")
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateJob enqueues a new ingestion job.
func (h *IngestionHandler) CreateJob(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	if user == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}
	if !rbac.HasPermission(user.Role, "ingestion_manage") {
		appErrors.RespondForbidden(c, "insufficient permissions")
		return
	}

	var req createJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondBadRequest(c, "invalid request body", err.Error())
		return
	}

	job, err := h.engine.EnqueueJob(ingestion.EnqueueRequest{
		ImportSourceID:  req.ImportSourceID,
		Priority:        req.Priority,
		DependencyGroup: req.DependencyGroup,
		Mode:            req.Mode,
	})
	if err != nil {
		appErrors.RespondBadRequest(c, err.Error(), nil)
		return
	}

	audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: &user.ID,
		ActionType:  audit.ActionCreate,
		TargetType:  "ingestion_job",
		TargetID:    fmt.Sprintf("%d", job.ID),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
	})

	c.JSON(http.StatusCreated, job)
}

// GetJob returns a single ingestion job.
func (h *IngestionHandler) GetJob(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	if user == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}
	if !rbac.HasPermission(user.Role, "ingestion_view") {
		appErrors.RespondForbidden(c, "insufficient permissions")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid job ID", nil)
		return
	}

	var job models.IngestionJob
	if err := h.db.First(&job, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "ingestion job not found")
			return
		}
		appErrors.RespondInternalError(c, "failed to get ingestion job")
		return
	}

	c.JSON(http.StatusOK, job)
}

// RetryJob retries a failed ingestion job.
func (h *IngestionHandler) RetryJob(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	if user == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}
	if !rbac.HasPermission(user.Role, "ingestion_manage") {
		appErrors.RespondForbidden(c, "insufficient permissions")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid job ID", nil)
		return
	}

	var job models.IngestionJob
	if err := h.db.First(&job, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "ingestion job not found")
			return
		}
		appErrors.RespondInternalError(c, "failed to get ingestion job")
		return
	}

	if err := h.engine.RetryJob(&job); err != nil {
		appErrors.RespondBadRequest(c, err.Error(), nil)
		return
	}

	audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: &user.ID,
		ActionType:  audit.ActionUpdate,
		TargetType:  "ingestion_job",
		TargetID:    fmt.Sprintf("%d", job.ID),
		MetadataJSON: `{"action":"retry"}`,
		IPAddress:   c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
	})

	c.JSON(http.StatusOK, job)
}

// AcknowledgeJob handles operator acknowledgment for failed jobs.
func (h *IngestionHandler) AcknowledgeJob(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	if user == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}
	if !rbac.HasPermission(user.Role, "ingestion_manage") {
		appErrors.RespondForbidden(c, "insufficient permissions")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid job ID", nil)
		return
	}

	var req acknowledgeJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondBadRequest(c, "invalid request body", err.Error())
		return
	}

	if err := h.engine.AcknowledgeJob(uint(id), user.ID, req.Reason); err != nil {
		appErrors.RespondBadRequest(c, err.Error(), nil)
		return
	}

	audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID:  &user.ID,
		ActionType:   audit.ActionApprove,
		TargetType:   "ingestion_job",
		TargetID:     fmt.Sprintf("%d", id),
		MetadataJSON: fmt.Sprintf(`{"action":"acknowledge","reason":%q}`, req.Reason),
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
	})

	// Reload job to return updated state.
	var job models.IngestionJob
	h.db.First(&job, id)

	c.JSON(http.StatusOK, job)
}

// ListCheckpoints returns checkpoints for a job.
func (h *IngestionHandler) ListCheckpoints(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	if user == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}
	if !rbac.HasPermission(user.Role, "ingestion_view") {
		appErrors.RespondForbidden(c, "insufficient permissions")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid job ID", nil)
		return
	}

	// Verify job exists.
	var job models.IngestionJob
	if err := h.db.First(&job, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "ingestion job not found")
			return
		}
		appErrors.RespondInternalError(c, "failed to get ingestion job")
		return
	}

	var checkpoints []models.IngestionCheckpoint
	if err := h.db.Where("job_id = ?", id).
		Order("created_at ASC").
		Find(&checkpoints).Error; err != nil {
		appErrors.RespondInternalError(c, "failed to list checkpoints")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id":      id,
		"checkpoints": checkpoints,
		"count":       len(checkpoints),
	})
}

// ListFailures returns failures for a job.
func (h *IngestionHandler) ListFailures(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	if user == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}
	if !rbac.HasPermission(user.Role, "ingestion_view") {
		appErrors.RespondForbidden(c, "insufficient permissions")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid job ID", nil)
		return
	}

	var job models.IngestionJob
	if err := h.db.First(&job, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "ingestion job not found")
			return
		}
		appErrors.RespondInternalError(c, "failed to get ingestion job")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	query := h.db.Model(&models.IngestionFailure{}).Where("job_id = ?", id)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		appErrors.RespondInternalError(c, "failed to count failures")
		return
	}

	offset := (page - 1) * pageSize
	var failures []models.IngestionFailure
	if err := query.Order("record_index ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&failures).Error; err != nil {
		appErrors.RespondInternalError(c, "failed to list failures")
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id":      id,
		"failures":    failures,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}
