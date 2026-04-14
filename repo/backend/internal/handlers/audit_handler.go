package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/internal/audit"
	"backend/internal/auth"
	appErrors "backend/internal/errors"
	"backend/internal/logging"
	"backend/internal/models"
)

// AuditHandler handles audit log and audit delete request endpoints.
type AuditHandler struct {
	db *gorm.DB
}

// NewAuditHandler creates a new AuditHandler instance.
func NewAuditHandler(db *gorm.DB) *AuditHandler {
	return &AuditHandler{db: db}
}

// RegisterRoutes registers the audit routes on the given router group.
func (h *AuditHandler) RegisterRoutes(rg *gin.RouterGroup) {
	auditGroup := rg.Group("/audit")
	{
		auditGroup.GET("/logs", h.ListLogs)
		auditGroup.GET("/logs/:id", h.GetLog)
		auditGroup.GET("/logs/search", h.SearchLogs)

		auditGroup.GET("/delete-requests", h.ListDeleteRequests)
		auditGroup.POST("/delete-requests", h.CreateDeleteRequest)
		auditGroup.GET("/delete-requests/:id", h.GetDeleteRequest)
		auditGroup.POST("/delete-requests/:id/approve", h.ApproveDeleteRequest)
		auditGroup.POST("/delete-requests/:id/execute", h.ExecuteDeleteRequest)
	}
}

// ListLogs returns paginated audit logs with optional filtering.
func (h *AuditHandler) ListLogs(c *gin.Context) {
	filter := audit.AuditFilter{
		ActionType: c.Query("action_type"),
		TargetType: c.Query("target_type"),
	}

	// Parse actor_id
	if actorIDStr := c.Query("actor_id"); actorIDStr != "" {
		actorID, err := strconv.ParseUint(actorIDStr, 10, 32)
		if err != nil {
			appErrors.RespondBadRequest(c, "invalid actor_id parameter", nil)
			return
		}
		uid := uint(actorID)
		filter.ActorID = &uid
	}

	// Parse date range
	if startStr := c.Query("start_date"); startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			appErrors.RespondBadRequest(c, "invalid start_date format, use RFC3339", nil)
			return
		}
		filter.StartDate = &t
	}
	if endStr := c.Query("end_date"); endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			appErrors.RespondBadRequest(c, "invalid end_date format, use RFC3339", nil)
			return
		}
		filter.EndDate = &t
	}

	// Parse pagination
	filter.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	filter.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "50"))

	logs, total, err := audit.GetAuditLogs(h.db, filter)
	if err != nil {
		logging.Error("audit", "list_logs", fmt.Sprintf("failed to list audit logs: %v", err))
		appErrors.RespondInternalError(c, "failed to retrieve audit logs")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      logs,
		"total":     total,
		"page":      filter.Page,
		"page_size": filter.PageSize,
	})
}

// GetLog returns a single audit log entry by ID.
func (h *AuditHandler) GetLog(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid audit log ID", nil)
		return
	}

	var logEntry models.AuditLog
	result := h.db.First(&logEntry, uint(id))
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "audit log not found")
			return
		}
		logging.Error("audit", "get_log", fmt.Sprintf("database error: %v", result.Error))
		appErrors.RespondInternalError(c, "failed to retrieve audit log")
		return
	}

	c.JSON(http.StatusOK, logEntry)
}

// SearchLogs performs a filtered search of audit logs (same as ListLogs, provided for route compatibility).
func (h *AuditHandler) SearchLogs(c *gin.Context) {
	h.ListLogs(c)
}

// ListDeleteRequests returns all audit delete requests.
func (h *AuditHandler) ListDeleteRequests(c *gin.Context) {
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

	var total int64
	h.db.Model(&models.AuditDeleteRequest{}).Count(&total)

	var requests []models.AuditDeleteRequest
	offset := (page - 1) * pageSize
	result := h.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&requests)
	if result.Error != nil {
		logging.Error("audit", "list_delete_requests", fmt.Sprintf("database error: %v", result.Error))
		appErrors.RespondInternalError(c, "failed to retrieve delete requests")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      requests,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// createDeleteRequestBody represents the JSON body for creating a delete request.
type createDeleteRequestBody struct {
	Reason     string `json:"reason" binding:"required"`
	TargetType string `json:"target_type" binding:"required"`
	TargetID   string `json:"target_id" binding:"required"`
}

// CreateDeleteRequest creates a new audit delete request. Requires admin role.
func (h *AuditHandler) CreateDeleteRequest(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	if user == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}

	var body createDeleteRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		appErrors.RespondBadRequest(c, "invalid request body", err.Error())
		return
	}

	request := models.AuditDeleteRequest{
		RequestedBy: user.ID,
		Reason:      body.Reason,
		State:       "pending",
		TargetType:  body.TargetType,
		TargetID:    body.TargetID,
	}

	if err := h.db.Create(&request).Error; err != nil {
		logging.Error("audit", "create_delete_request", fmt.Sprintf("database error: %v", err))
		appErrors.RespondInternalError(c, "failed to create delete request")
		return
	}

	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: &user.ID,
		ActionType:  audit.ActionCreate,
		TargetType:  "audit_delete_request",
		TargetID:    fmt.Sprintf("%d", request.ID),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	})

	c.JSON(http.StatusCreated, request)
}

// GetDeleteRequest returns a single audit delete request by ID.
func (h *AuditHandler) GetDeleteRequest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid delete request ID", nil)
		return
	}

	var request models.AuditDeleteRequest
	result := h.db.First(&request, uint(id))
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "delete request not found")
			return
		}
		logging.Error("audit", "get_delete_request", fmt.Sprintf("database error: %v", result.Error))
		appErrors.RespondInternalError(c, "failed to retrieve delete request")
		return
	}

	c.JSON(http.StatusOK, request)
}

// ApproveDeleteRequest approves an audit delete request with dual-approval enforcement.
// The same user cannot be both requester and approver, and cannot approve twice.
func (h *AuditHandler) ApproveDeleteRequest(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	if user == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid delete request ID", nil)
		return
	}

	var request models.AuditDeleteRequest
	result := h.db.First(&request, uint(id))
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "delete request not found")
			return
		}
		logging.Error("audit", "approve_delete_request", fmt.Sprintf("database error: %v", result.Error))
		appErrors.RespondInternalError(c, "failed to retrieve delete request")
		return
	}

	// Check the request is still pending or partially approved
	if request.State != "pending" && request.State != "partially_approved" {
		appErrors.RespondConflict(c, fmt.Sprintf("delete request is in state '%s' and cannot be approved", request.State), nil)
		return
	}

	// Requester cannot approve their own request
	if request.RequestedBy == user.ID {
		appErrors.RespondForbidden(c, "the requester cannot approve their own delete request")
		return
	}

	// First approval
	if request.ApproverOne == nil {
		request.ApproverOne = &user.ID
		request.State = "partially_approved"
		if err := h.db.Model(&request).Updates(map[string]interface{}{
			"approver_one": user.ID,
			"state":        "partially_approved",
		}).Error; err != nil {
			logging.Error("audit", "approve_delete_request", fmt.Sprintf("database error: %v", err))
			appErrors.RespondInternalError(c, "failed to approve delete request")
			return
		}

		_ = audit.LogAction(h.db, audit.AuditEntry{
			ActorUserID: &user.ID,
			ActionType:  audit.ActionApprove,
			TargetType:  "audit_delete_request",
			TargetID:    fmt.Sprintf("%d", request.ID),
			MetadataJSON: `{"approval":"first"}`,
			IPAddress:   c.ClientIP(),
			UserAgent:   c.Request.UserAgent(),
		})

		c.JSON(http.StatusOK, gin.H{
			"message": "first approval recorded, awaiting second approval",
			"data":    request,
		})
		return
	}

	// Second approval: must be a different user than the first approver
	if *request.ApproverOne == user.ID {
		appErrors.RespondForbidden(c, "the same user cannot provide both approvals")
		return
	}

	request.ApproverTwo = &user.ID
	request.State = "approved"
	if err := h.db.Model(&request).Updates(map[string]interface{}{
		"approver_two": user.ID,
		"state":        "approved",
	}).Error; err != nil {
		logging.Error("audit", "approve_delete_request", fmt.Sprintf("database error: %v", err))
		appErrors.RespondInternalError(c, "failed to approve delete request")
		return
	}

	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: &user.ID,
		ActionType:  audit.ActionApprove,
		TargetType:  "audit_delete_request",
		TargetID:    fmt.Sprintf("%d", request.ID),
		MetadataJSON: `{"approval":"second"}`,
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "delete request fully approved and ready for execution",
		"data":    request,
	})
}

// ExecuteDeleteRequest carries out an approved audit deletion.
func (h *AuditHandler) ExecuteDeleteRequest(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	if user == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid delete request ID", nil)
		return
	}

	var request models.AuditDeleteRequest
	result := h.db.First(&request, uint(id))
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "delete request not found")
			return
		}
		logging.Error("audit", "execute_delete_request", fmt.Sprintf("database error: %v", result.Error))
		appErrors.RespondInternalError(c, "failed to retrieve delete request")
		return
	}

	if request.State != "approved" {
		appErrors.RespondConflict(c, "delete request must be fully approved before execution", nil)
		return
	}

	// Execute the deletion
	deleteResult := h.db.Where("target_type = ? AND target_id = ?", request.TargetType, request.TargetID).
		Delete(&models.AuditLog{})
	if deleteResult.Error != nil {
		logging.Error("audit", "execute_delete_request", fmt.Sprintf("failed to delete audit logs: %v", deleteResult.Error))
		appErrors.RespondInternalError(c, "failed to execute deletion")
		return
	}

	now := time.Now()
	h.db.Model(&request).Updates(map[string]interface{}{
		"state":       "executed",
		"executed_at": now,
	})

	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID:  &user.ID,
		ActionType:   audit.ActionDelete,
		TargetType:   "audit_delete_request",
		TargetID:     fmt.Sprintf("%d", request.ID),
		MetadataJSON: fmt.Sprintf(`{"deleted_count":%d}`, deleteResult.RowsAffected),
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, gin.H{
		"message":       "audit logs deleted successfully",
		"deleted_count": deleteResult.RowsAffected,
	})
}
