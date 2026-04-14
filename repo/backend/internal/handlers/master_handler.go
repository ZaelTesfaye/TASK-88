package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/internal/audit"
	"backend/internal/auth"
	appErrors "backend/internal/errors"
	"backend/internal/masterdata"
	"backend/internal/models"
	"backend/internal/org"
	"backend/internal/rbac"
)

// MasterHandler handles master data endpoints.
type MasterHandler struct {
	db         *gorm.DB
	service    *masterdata.MasterService
	orgService *org.OrgService
}

// NewMasterHandler creates a new MasterHandler.
func NewMasterHandler(db *gorm.DB) *MasterHandler {
	return &MasterHandler{
		db:         db,
		service:    masterdata.NewMasterService(db),
		orgService: org.NewOrgService(db),
	}
}

// RegisterRoutes registers all master data routes.
func (h *MasterHandler) RegisterRoutes(rg *gin.RouterGroup) {
	master := rg.Group("/master")
	{
		master.GET("/:entity", h.ListRecords)
		master.POST("/:entity", h.CreateRecord)
		master.GET("/:entity/duplicates", h.CheckDuplicates)
		master.GET("/:entity/:id", h.GetRecord)
		master.PATCH("/:entity/:id", h.UpdateRecord)
		master.PUT("/:entity/:id", h.UpdateRecord)
		master.POST("/:entity/:id/deactivate", h.DeactivateRecord)
		master.POST("/:entity/import", h.ImportRecords)
		master.GET("/:entity/:id/history", h.GetRecordHistory)
	}
}

// ListRecords returns paginated, filtered, sorted records for an entity type.
func (h *MasterHandler) ListRecords(c *gin.Context) {
	entityType := c.Param("entity")

	filter := masterdata.ListFilter{
		Search:    c.Query("search"),
		Status:    c.DefaultQuery("status", "active"),
		SortBy:    c.DefaultQuery("sort_by", "natural_key"),
		SortOrder: c.DefaultQuery("sort_order", "asc"),
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "25"))
	filter.Page = page
	filter.PageSize = pageSize

	// Apply scope filtering from user context.
	user := auth.GetCurrentUser(c)
	if user != nil {
		filter.CityScope = user.CityScope
		filter.DeptScope = user.DepartmentScope

		_, scopeIDs, err := h.orgService.GetUserContext(user.ID)
		if err == nil && len(scopeIDs) > 0 {
			filter.NodeIDs = scopeIDs
		}
	}

	result, err := h.service.ListRecords(entityType, filter)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

// CreateRecord creates a new master record.
func (h *MasterHandler) CreateRecord(c *gin.Context) {
	entityType := c.Param("entity")

	var req masterdata.CreateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondBadRequest(c, "invalid request body", err.Error())
		return
	}

	record, err := h.service.CreateRecord(entityType, req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	// Audit log.
	userID := getCurrentUserID(c)
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: userID,
		ActionType:  audit.ActionCreate,
		TargetType:  fmt.Sprintf("master_%s", entityType),
		TargetID:    fmt.Sprintf("%d", record.ID),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	})

	c.JSON(http.StatusCreated, gin.H{
		"data": record,
	})
}

// GetRecord returns a single master record by ID.
func (h *MasterHandler) GetRecord(c *gin.Context) {
	entityType := c.Param("entity")
	idStr := c.Param("id")

	// Check if idStr is "duplicates" to handle route ambiguity.
	if idStr == "duplicates" {
		h.CheckDuplicates(c)
		return
	}
	// Check if idStr is "import" to handle route ambiguity.
	if idStr == "import" {
		h.ImportRecords(c)
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, fmt.Sprintf("invalid id parameter: %s", idStr), nil)
		return
	}

	var record struct {
		ID          uint   `json:"id"`
		EntityType  string `json:"entity_type"`
		NaturalKey  string `json:"natural_key"`
		PayloadJSON string `json:"payload_json"`
		Status      string `json:"status"`
	}

	result := h.db.Table("master_records").Where("id = ? AND entity_type = ?", uint(id), entityType).Scan(&record)
	if result.Error != nil {
		appErrors.RespondInternalError(c, "failed to load record")
		return
	}
	if result.RowsAffected == 0 {
		appErrors.RespondNotFound(c, fmt.Sprintf("%s record %d not found", entityType, id))
		return
	}

	// Object-level scope check.
	if !h.isRecordInScope(c, uint(id)) {
		appErrors.RespondForbidden(c, "access denied: record is outside your organizational scope")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": record,
	})
}

// UpdateRecord updates an existing master record.
func (h *MasterHandler) UpdateRecord(c *gin.Context) {
	entityType := c.Param("entity")
	id, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	// Verify entity type matches.
	var existing struct {
		EntityType string
	}
	if result := h.db.Table("master_records").Select("entity_type").Where("id = ?", id).Scan(&existing); result.RowsAffected == 0 {
		appErrors.RespondNotFound(c, fmt.Sprintf("master record %d not found", id))
		return
	}
	if existing.EntityType != entityType {
		appErrors.RespondBadRequest(c, fmt.Sprintf("record %d is not a %s record", id, entityType), nil)
		return
	}

	// Object-level scope check.
	if !h.isRecordInScope(c, id) {
		appErrors.RespondForbidden(c, "access denied: record is outside your organizational scope")
		return
	}

	var req masterdata.UpdateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondBadRequest(c, "invalid request body", err.Error())
		return
	}

	record, svcErr := h.service.UpdateRecord(id, req)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	// Audit log.
	userID := getCurrentUserID(c)
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: userID,
		ActionType:  audit.ActionUpdate,
		TargetType:  fmt.Sprintf("master_%s", entityType),
		TargetID:    fmt.Sprintf("%d", record.ID),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, gin.H{
		"data": record,
	})
}

// DeactivateRecord soft-deactivates a record with a reason.
func (h *MasterHandler) DeactivateRecord(c *gin.Context) {
	entityType := c.Param("entity")
	id, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondBadRequest(c, "invalid request body: reason is required", err.Error())
		return
	}

	// Object-level scope check.
	if !h.isRecordInScope(c, id) {
		appErrors.RespondForbidden(c, "access denied: record is outside your organizational scope")
		return
	}

	userID := getCurrentUserID(c)
	var actorID uint
	if userID != nil {
		actorID = *userID
	}

	if svcErr := h.service.DeactivateRecord(id, req.Reason, actorID); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	// Audit log.
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: userID,
		ActionType:  audit.ActionDeactivate,
		TargetType:  fmt.Sprintf("master_%s", entityType),
		TargetID:    fmt.Sprintf("%d", id),
		MetadataJSON: fmt.Sprintf(`{"reason":"%s"}`, strings.ReplaceAll(req.Reason, `"`, `\"`)),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("%s record %d deactivated", entityType, id),
	})
}

// CheckDuplicates checks for duplicate records by key.
func (h *MasterHandler) CheckDuplicates(c *gin.Context) {
	entityType := c.Param("entity")
	key := c.Query("key")

	if key == "" {
		appErrors.RespondBadRequest(c, "query parameter 'key' is required", nil)
		return
	}

	records, err := h.service.CheckDuplicates(entityType, key)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": records,
	})
}

// ImportRecords handles bulk CSV/XLSX import via multipart file upload.
func (h *MasterHandler) ImportRecords(c *gin.Context) {
	entityType := c.Param("entity")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		appErrors.RespondBadRequest(c, "file upload is required (form field: 'file')", err.Error())
		return
	}
	defer file.Close()

	// Check file size.
	if header.Size > 50*1024*1024 {
		appErrors.RespondValidationError(c, "file size exceeds maximum of 50MB", nil)
		return
	}

	fileData, err := io.ReadAll(file)
	if err != nil {
		appErrors.RespondInternalError(c, fmt.Sprintf("failed to read uploaded file: %v", err))
		return
	}

	result, svcErr := h.service.ImportRecords(entityType, fileData, header.Filename)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	// Audit log.
	userID := getCurrentUserID(c)
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID:  userID,
		ActionType:   audit.ActionImport,
		TargetType:   fmt.Sprintf("master_%s", entityType),
		TargetID:     header.Filename,
		MetadataJSON: fmt.Sprintf(`{"total_rows":%d,"success":%d,"errors":%d}`, result.TotalRows, result.SuccessCount, result.ErrorCount),
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
	})

	statusCode := http.StatusOK
	if result.ErrorCount > 0 && result.SuccessCount > 0 {
		statusCode = http.StatusOK // Partial success.
	} else if result.ErrorCount > 0 && result.SuccessCount == 0 {
		statusCode = http.StatusUnprocessableEntity
	} else {
		statusCode = http.StatusCreated
	}

	c.JSON(statusCode, gin.H{
		"data": result,
	})
}

// GetRecordHistory returns the deactivation history for a record.
func (h *MasterHandler) GetRecordHistory(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	type DeactivationEventResponse struct {
		ID          uint   `json:"id"`
		RecordID    uint   `json:"record_id"`
		Reason      string `json:"reason"`
		ActorUserID uint   `json:"actor_user_id"`
	}

	var events []DeactivationEventResponse
	if err := h.db.Table("deactivation_events").Where("record_id = ?", id).
		Order("created_at DESC").Scan(&events).Error; err != nil {
		appErrors.RespondInternalError(c, "failed to load record history")
		return
	}

	if events == nil {
		events = []DeactivationEventResponse{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": events,
	})
}

// isRecordInScope checks whether the current user's scope covers the given
// master record. SystemAdmins always pass. Fail-closed: a scoped role with
// no context assignment is denied.
func (h *MasterHandler) isRecordInScope(c *gin.Context, recordID uint) bool {
	user := auth.GetCurrentUser(c)
	if user == nil {
		return false
	}
	if user.Role == rbac.SystemAdmin {
		return true
	}

	// Fail-closed: non-admin with no scope dimensions at all → deny.
	if user.CityScope == "" && user.DepartmentScope == "" {
		return false
	}

	_, scopeIDs, err := h.orgService.GetUserContext(user.ID)
	if err != nil || len(scopeIDs) == 0 {
		// No context assignment — fail-closed for scoped roles.
		return false
	}

	// Check if this record is linked to any version within the user's scope.
	scopeKeys := make([]string, len(scopeIDs))
	for i, id := range scopeIDs {
		scopeKeys[i] = fmt.Sprintf("node:%d", id)
	}

	var inScopeCount int64
	h.db.Model(&models.MasterVersionItem{}).
		Joins("JOIN master_versions ON master_versions.id = master_version_items.version_id").
		Where("master_version_items.master_record_id = ? AND master_versions.scope_key IN ?", recordID, scopeKeys).
		Count(&inScopeCount)
	if inScopeCount > 0 {
		return true
	}

	// If the record has no version links at all, it's unscoped – allow access.
	var totalLinks int64
	h.db.Model(&models.MasterVersionItem{}).Where("master_record_id = ?", recordID).Count(&totalLinks)
	return totalLinks == 0
}
