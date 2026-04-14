package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/internal/audit"
	"backend/internal/auth"
	appErrors "backend/internal/errors"
	"backend/internal/masterdata"
)

// VersionHandler handles master data version endpoints.
type VersionHandler struct {
	db      *gorm.DB
	service *masterdata.VersionService
}

// NewVersionHandler creates a new VersionHandler.
func NewVersionHandler(db *gorm.DB) *VersionHandler {
	return &VersionHandler{
		db:      db,
		service: masterdata.NewVersionService(db),
	}
}

// RegisterRoutes registers all version-related routes.
func (h *VersionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	versions := rg.Group("/versions")
	{
		versions.GET("/:entity", h.ListVersions)
		versions.POST("/:entity/draft", h.CreateDraft)
		versions.GET("/:entity/:id", h.GetVersion)
		versions.POST("/:entity/:id/submit-review", h.SubmitForReview)
		versions.POST("/:entity/:id/activate", h.ActivateVersion)
		versions.POST("/:entity/:id/rollback", h.RollbackVersion)
		versions.GET("/:entity/:id/items", h.ListVersionItems)
		versions.POST("/:entity/:id/items", h.AddVersionItems)
		versions.DELETE("/:entity/:id/items/:itemId", h.RemoveVersionItem)
		versions.GET("/:entity/:id/diff", h.DiffVersions)
	}
}

// ListVersions returns version history for an entity type.
func (h *VersionHandler) ListVersions(c *gin.Context) {
	entityType := c.Param("entity")
	scopeKey := c.Query("scope_key")

	versions, err := h.service.GetVersionHistory(entityType, scopeKey)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": versions,
	})
}

// CreateDraft creates a new draft version.
func (h *VersionHandler) CreateDraft(c *gin.Context) {
	entityType := c.Param("entity")

	var req struct {
		ScopeKey string `json:"scope_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondBadRequest(c, "invalid request body: scope_key is required", err.Error())
		return
	}

	user := auth.GetCurrentUser(c)
	var createdBy uint
	if user != nil {
		createdBy = user.ID
	}

	version, err := h.service.CreateDraft(entityType, req.ScopeKey, createdBy)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	// Audit log.
	userID := getCurrentUserID(c)
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: userID,
		ActionType:  audit.ActionCreate,
		TargetType:  fmt.Sprintf("version_%s", entityType),
		TargetID:    fmt.Sprintf("%d", version.ID),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	})

	c.JSON(http.StatusCreated, gin.H{
		"data": version,
	})
}

// GetVersion returns a single version by ID.
func (h *VersionHandler) GetVersion(c *gin.Context) {
	idStr := c.Param("id")

	// Handle route ambiguity: if idStr is "draft", redirect to CreateDraft.
	if idStr == "draft" {
		h.CreateDraft(c)
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, fmt.Sprintf("invalid id parameter: %s", idStr), nil)
		return
	}

	version, svcErr := h.service.GetVersion(uint(id))
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": version,
	})
}

// SubmitForReview transitions a draft version to review.
func (h *VersionHandler) SubmitForReview(c *gin.Context) {
	entityType := c.Param("entity")
	id, err := parseVersionIDParam(c)
	if err != nil {
		return
	}

	var req struct {
		ReviewerID uint `json:"reviewer_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondBadRequest(c, "invalid request body: reviewer_id is required", err.Error())
		return
	}

	version, svcErr := h.service.SubmitForReview(id, req.ReviewerID)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	// Audit log.
	userID := getCurrentUserID(c)
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID:  userID,
		ActionType:   audit.ActionUpdate,
		TargetType:   fmt.Sprintf("version_%s", entityType),
		TargetID:     fmt.Sprintf("%d", version.ID),
		MetadataJSON: fmt.Sprintf(`{"action":"submit_for_review","reviewer_id":%d}`, req.ReviewerID),
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, gin.H{
		"data": version,
	})
}

// ActivateVersion transitions a reviewed version to active.
func (h *VersionHandler) ActivateVersion(c *gin.Context) {
	entityType := c.Param("entity")
	id, err := parseVersionIDParam(c)
	if err != nil {
		return
	}

	user := auth.GetCurrentUser(c)
	var activatorID uint
	if user != nil {
		activatorID = user.ID
	}

	version, svcErr := h.service.Activate(id, activatorID)
	if svcErr != nil {
		// Check for conflict (concurrent activation).
		if appErr, ok := svcErr.(*appErrors.AppError); ok && appErr.Code == "CONFLICT" {
			appErrors.RespondWithError(c, http.StatusConflict, appErr)
			return
		}
		handleServiceError(c, svcErr)
		return
	}

	// Audit log.
	userID := getCurrentUserID(c)
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID:  userID,
		ActionType:   audit.ActionActivate,
		TargetType:   fmt.Sprintf("version_%s", entityType),
		TargetID:     fmt.Sprintf("%d", version.ID),
		MetadataJSON: fmt.Sprintf(`{"action":"activate","version_no":%d}`, version.VersionNo),
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, gin.H{
		"data": version,
	})
}

// RollbackVersion reverts to a previous version.
func (h *VersionHandler) RollbackVersion(c *gin.Context) {
	entityType := c.Param("entity")
	id, err := parseVersionIDParam(c)
	if err != nil {
		return
	}

	user := auth.GetCurrentUser(c)
	var actorID uint
	if user != nil {
		actorID = user.ID
	}

	version, svcErr := h.service.Rollback(id, actorID)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	// Audit log.
	userID := getCurrentUserID(c)
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID:  userID,
		ActionType:   audit.ActionUpdate,
		TargetType:   fmt.Sprintf("version_%s", entityType),
		TargetID:     fmt.Sprintf("%d", version.ID),
		MetadataJSON: fmt.Sprintf(`{"action":"rollback","version_no":%d}`, version.VersionNo),
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, gin.H{
		"data": version,
	})
}

// ListVersionItems returns all items in a version.
func (h *VersionHandler) ListVersionItems(c *gin.Context) {
	id, err := parseVersionIDParam(c)
	if err != nil {
		return
	}

	items, svcErr := h.service.GetVersionItems(id)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": items,
	})
}

// AddVersionItems adds records to a version.
func (h *VersionHandler) AddVersionItems(c *gin.Context) {
	entityType := c.Param("entity")
	id, err := parseVersionIDParam(c)
	if err != nil {
		return
	}

	var req struct {
		RecordIDs []uint `json:"record_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondBadRequest(c, "invalid request body: record_ids is required", err.Error())
		return
	}

	if svcErr := h.service.AddItems(id, req.RecordIDs); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	// Audit log.
	userID := getCurrentUserID(c)
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID:  userID,
		ActionType:   audit.ActionUpdate,
		TargetType:   fmt.Sprintf("version_%s", entityType),
		TargetID:     fmt.Sprintf("%d", id),
		MetadataJSON: fmt.Sprintf(`{"action":"add_items","count":%d}`, len(req.RecordIDs)),
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("%d items added to version %d", len(req.RecordIDs), id),
	})
}

// RemoveVersionItem removes a single item from a version.
func (h *VersionHandler) RemoveVersionItem(c *gin.Context) {
	entityType := c.Param("entity")
	versionID, err := parseVersionIDParam(c)
	if err != nil {
		return
	}

	itemIDStr := c.Param("itemId")
	itemID, parseErr := strconv.ParseUint(itemIDStr, 10, 32)
	if parseErr != nil {
		appErrors.RespondBadRequest(c, fmt.Sprintf("invalid itemId parameter: %s", itemIDStr), nil)
		return
	}

	if svcErr := h.service.RemoveItem(versionID, uint(itemID)); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	// Audit log.
	userID := getCurrentUserID(c)
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID:  userID,
		ActionType:   audit.ActionDelete,
		TargetType:   fmt.Sprintf("version_%s_item", entityType),
		TargetID:     fmt.Sprintf("%d", itemID),
		MetadataJSON: fmt.Sprintf(`{"version_id":%d}`, versionID),
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("item %d removed from version %d", itemID, versionID),
	})
}

// DiffVersions returns the diff between two versions.
func (h *VersionHandler) DiffVersions(c *gin.Context) {
	id, err := parseVersionIDParam(c)
	if err != nil {
		return
	}

	compareIDStr := c.Query("compare_to")
	if compareIDStr == "" {
		appErrors.RespondBadRequest(c, "query parameter 'compare_to' is required", nil)
		return
	}

	compareID, parseErr := strconv.ParseUint(compareIDStr, 10, 32)
	if parseErr != nil {
		appErrors.RespondBadRequest(c, fmt.Sprintf("invalid compare_to parameter: %s", compareIDStr), nil)
		return
	}

	diff, svcErr := h.service.DiffVersions(id, uint(compareID))
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": diff,
	})
}

// parseVersionIDParam parses the :id parameter from version routes.
func parseVersionIDParam(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, fmt.Sprintf("invalid id parameter: %s", idStr), nil)
		return 0, err
	}
	return uint(id), nil
}

// CreateVersion is an alias for CreateDraft, used by the router.
func (h *VersionHandler) CreateVersion(c *gin.Context) {
	h.CreateDraft(c)
}

// ReviewVersion is an alias for SubmitForReview, used by the router.
func (h *VersionHandler) ReviewVersion(c *gin.Context) {
	h.SubmitForReview(c)
}

// AddVersionItem is an alias for AddVersionItems (singular form), used by the router.
func (h *VersionHandler) AddVersionItem(c *gin.Context) {
	h.AddVersionItems(c)
}
