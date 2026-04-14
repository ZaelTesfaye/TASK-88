package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/internal/audit"
	appErrors "backend/internal/errors"
	"backend/internal/models"
	"backend/internal/security"
)

type SecurityHandler struct {
	db      *gorm.DB
	service *security.SecurityService
}

func NewSecurityHandler(db *gorm.DB) *SecurityHandler {
	return &SecurityHandler{
		db:      db,
		service: security.NewSecurityService(db),
	}
}

func (h *SecurityHandler) RegisterRoutes(rg *gin.RouterGroup) {
	sec := rg.Group("/security")
	{
		sec.GET("/sensitive-fields", h.ListSensitiveFields)
		sec.POST("/sensitive-fields", h.CreateSensitiveField)
		sec.PUT("/sensitive-fields/:id", h.UpdateSensitiveField)
		sec.DELETE("/sensitive-fields/:id", h.DeleteSensitiveField)

		sec.GET("/keys", h.ListKeys)
		sec.GET("/keys/:id", h.GetKey)
		sec.POST("/keys/rotate", h.RotateKey)

		sec.GET("/password-reset", h.ListPasswordResetRequests)
		sec.POST("/password-reset", h.RequestPasswordReset)
		sec.POST("/password-reset/:id/approve", h.ApprovePasswordReset)

		sec.GET("/retention-policies", h.ListRetentionPolicies)
		sec.POST("/retention-policies", h.CreateRetentionPolicy)
		sec.PUT("/retention-policies/:id", h.UpdateRetentionPolicy)

		sec.GET("/legal-holds", h.ListLegalHolds)
		sec.POST("/legal-holds", h.CreateLegalHold)
		sec.POST("/legal-holds/:id/release", h.ReleaseLegalHold)

		sec.POST("/purge-runs/dry-run", h.DryRunPurge)
		sec.POST("/purge-runs/execute", h.ExecutePurge)
		sec.GET("/purge-runs", h.ListPurgeRuns)
	}
}

func (h *SecurityHandler) ListSensitiveFields(c *gin.Context) {
	fields, err := h.service.GetSensitiveFields()
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": fields, "total": len(fields)})
}

func (h *SecurityHandler) CreateSensitiveField(c *gin.Context) {
	var field models.SensitiveFieldRegistry
	if err := c.ShouldBindJSON(&field); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}
	if err := h.service.CreateSensitiveField(&field); err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	if uid, exists := c.Get("user_id"); exists {
		userID := uid.(uint)
		_ = audit.LogAction(h.db, audit.AuditEntry{
			ActorUserID: &userID,
			ActionType:  audit.ActionCreate,
			TargetType:  "sensitive_field",
			TargetID:    field.FieldKey,
		})
	}
	c.JSON(http.StatusCreated, field)
}

func (h *SecurityHandler) UpdateSensitiveField(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid field ID", nil)
		return
	}
	var req security.UpdateFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}
	if err := h.service.UpdateSensitiveFieldByID(uint(id), req); err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "sensitive field not found")
			return
		}
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	if uid, exists := c.Get("user_id"); exists {
		userID := uid.(uint)
		_ = audit.LogAction(h.db, audit.AuditEntry{
			ActorUserID: &userID,
			ActionType:  audit.ActionUpdate,
			TargetType:  "sensitive_field",
			TargetID:    fmt.Sprintf("%d", id),
		})
	}
	c.JSON(http.StatusOK, gin.H{"message": "sensitive field updated"})
}

func (h *SecurityHandler) DeleteSensitiveField(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid field ID", nil)
		return
	}
	if err := h.service.DeleteSensitiveField(uint(id)); err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "sensitive field not found")
			return
		}
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	if uid, exists := c.Get("user_id"); exists {
		userID := uid.(uint)
		_ = audit.LogAction(h.db, audit.AuditEntry{
			ActorUserID: &userID,
			ActionType:  audit.ActionDelete,
			TargetType:  "sensitive_field",
			TargetID:    fmt.Sprintf("%d", id),
		})
	}
	c.JSON(http.StatusOK, gin.H{"message": "sensitive field deactivated"})
}

func (h *SecurityHandler) ListKeys(c *gin.Context) {
	keys, err := h.service.GetKeyRing()
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": keys, "total": len(keys)})
}

func (h *SecurityHandler) GetKey(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid key ID", nil)
		return
	}
	key, err := h.service.GetKeyByID(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "key not found")
			return
		}
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, key)
}

func (h *SecurityHandler) RotateKey(c *gin.Context) {
	var req struct {
		KeyID   string `json:"key_id" binding:"required"`
		Purpose string `json:"purpose" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}
	if err := h.service.RotateKey(req.KeyID, req.Purpose); err != nil {
		appErrors.RespondBadRequest(c, err.Error(), nil)
		return
	}
	if uid, exists := c.Get("user_id"); exists {
		userID := uid.(uint)
		_ = audit.LogAction(h.db, audit.AuditEntry{
			ActorUserID: &userID,
			ActionType:  audit.ActionUpdate,
			TargetType:  "key_ring",
			TargetID:    req.KeyID,
		})
	}
	c.JSON(http.StatusOK, gin.H{"message": "key rotated successfully"})
}

func (h *SecurityHandler) ListPasswordResetRequests(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	requests, total, err := h.service.GetPasswordResetRequests(page, pageSize)
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": requests, "total": total})
}

func (h *SecurityHandler) RequestPasswordReset(c *gin.Context) {
	var req struct {
		UserID uint   `json:"user_id" binding:"required"`
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}
	var userID uint
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(uint)
	}
	resetReq, err := h.service.CreatePasswordResetRequest(req.UserID, userID, req.Reason)
	if err != nil {
		appErrors.RespondBadRequest(c, err.Error(), nil)
		return
	}
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: &userID,
		ActionType:  audit.ActionCreate,
		TargetType:  "password_reset_request",
		TargetID:    fmt.Sprintf("%d", resetReq.ID),
	})
	c.JSON(http.StatusCreated, resetReq)
}

func (h *SecurityHandler) ApprovePasswordReset(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid request ID", nil)
		return
	}
	var userID uint
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(uint)
	}
	token, err := h.service.ApprovePasswordResetRequest(uint(id), userID)
	if err != nil {
		appErrors.RespondBadRequest(c, err.Error(), nil)
		return
	}
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: &userID,
		ActionType:  audit.ActionApprove,
		TargetType:  "password_reset_request",
		TargetID:    fmt.Sprintf("%d", id),
	})
	c.JSON(http.StatusOK, gin.H{"token": token, "message": "copy this token now - it will not be shown again", "expires_in": "1 hour"})
}

func (h *SecurityHandler) ListRetentionPolicies(c *gin.Context) {
	policies, err := h.service.GetRetentionPolicies()
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": policies, "total": len(policies)})
}

func (h *SecurityHandler) CreateRetentionPolicy(c *gin.Context) {
	var policy models.RetentionPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}
	if err := h.service.CreateRetentionPolicy(&policy); err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	if uid, exists := c.Get("user_id"); exists {
		userID := uid.(uint)
		_ = audit.LogAction(h.db, audit.AuditEntry{
			ActorUserID: &userID,
			ActionType:  audit.ActionCreate,
			TargetType:  "retention_policy",
			TargetID:    fmt.Sprintf("%d", policy.ID),
		})
	}
	c.JSON(http.StatusCreated, policy)
}

func (h *SecurityHandler) UpdateRetentionPolicy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid policy ID", nil)
		return
	}
	var req security.UpdateRetentionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}
	if err := h.service.UpdateRetentionPolicy(uint(id), req); err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "retention policy not found")
			return
		}
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	if uid, exists := c.Get("user_id"); exists {
		userID := uid.(uint)
		_ = audit.LogAction(h.db, audit.AuditEntry{
			ActorUserID: &userID,
			ActionType:  audit.ActionUpdate,
			TargetType:  "retention_policy",
			TargetID:    fmt.Sprintf("%d", id),
		})
	}
	c.JSON(http.StatusOK, gin.H{"message": "retention policy updated"})
}

func (h *SecurityHandler) ListLegalHolds(c *gin.Context) {
	holds, err := h.service.GetLegalHolds()
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": holds, "total": len(holds)})
}

func (h *SecurityHandler) CreateLegalHold(c *gin.Context) {
	var req security.CreateLegalHoldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}
	var userID uint
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(uint)
	}
	hold, err := h.service.CreateLegalHold(req, userID)
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: &userID,
		ActionType:  audit.ActionCreate,
		TargetType:  "legal_hold",
		TargetID:    fmt.Sprintf("%d", hold.ID),
	})
	c.JSON(http.StatusCreated, hold)
}

func (h *SecurityHandler) ReleaseLegalHold(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid hold ID", nil)
		return
	}
	var userID uint
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(uint)
	}
	if err := h.service.ReleaseLegalHold(uint(id), userID); err != nil {
		appErrors.RespondBadRequest(c, err.Error(), nil)
		return
	}
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: &userID,
		ActionType:  audit.ActionUpdate,
		TargetType:  "legal_hold",
		TargetID:    fmt.Sprintf("%d", id),
	})
	c.JSON(http.StatusOK, gin.H{"message": "legal hold released"})
}

func (h *SecurityHandler) DryRunPurge(c *gin.Context) {
	var req struct {
		ArtifactType string `json:"artifact_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}
	preview, err := h.service.DryRunPurge(req.ArtifactType)
	if err != nil {
		appErrors.RespondBadRequest(c, err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, preview)
}

func (h *SecurityHandler) ExecutePurge(c *gin.Context) {
	var req struct {
		ArtifactType string `json:"artifact_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}
	var userID uint
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(uint)
	}
	run, err := h.service.ExecutePurge(req.ArtifactType, userID)
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: &userID,
		ActionType:  audit.ActionDelete,
		TargetType:  "purge_run",
		TargetID:    fmt.Sprintf("%d", run.ID),
	})
	c.JSON(http.StatusCreated, run)
}

func (h *SecurityHandler) ListPurgeRuns(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	runs, total, err := h.service.GetPurgeRuns(page, pageSize)
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": runs, "total": total})
}
