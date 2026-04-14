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
	"backend/internal/org"
)

// OrgHandler handles org tree and context endpoints.
type OrgHandler struct {
	db      *gorm.DB
	service *org.OrgService
}

// NewOrgHandler creates a new OrgHandler.
func NewOrgHandler(db *gorm.DB) *OrgHandler {
	return &OrgHandler{
		db:      db,
		service: org.NewOrgService(db),
	}
}

// RegisterRoutes registers all org-related routes.
func (h *OrgHandler) RegisterRoutes(rg *gin.RouterGroup) {
	orgGroup := rg.Group("/org")
	{
		orgGroup.GET("/tree", h.GetTree)
		orgGroup.GET("/nodes", h.ListNodes)
		orgGroup.POST("/nodes", h.CreateNode)
		orgGroup.GET("/nodes/:id", h.GetNode)
		orgGroup.PATCH("/nodes/:id", h.UpdateNode)
		orgGroup.PUT("/nodes/:id", h.UpdateNode)
		orgGroup.DELETE("/nodes/:id", h.DeleteNode)
	}

	ctx := rg.Group("/context")
	{
		ctx.POST("/switch", h.SwitchContext)
		ctx.GET("/current", h.GetCurrentContext)
	}
}

// GetTree returns the full org tree as nested JSON.
func (h *OrgHandler) GetTree(c *gin.Context) {
	tree, err := h.service.GetTree()
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": tree,
	})
}

// ListNodes returns a flat list of all active org nodes.
func (h *OrgHandler) ListNodes(c *gin.Context) {
	tree, err := h.service.GetTree()
	if err != nil {
		handleServiceError(c, err)
		return
	}

	// Flatten the tree.
	var nodes []org.OrgTreeNode
	var flatten func(items []org.OrgTreeNode)
	flatten = func(items []org.OrgTreeNode) {
		for _, item := range items {
			nodes = append(nodes, org.OrgTreeNode{OrgNode: item.OrgNode, Children: nil})
			flatten(item.Children)
		}
	}
	flatten(tree)

	c.JSON(http.StatusOK, gin.H{
		"data": nodes,
	})
}

// CreateNode creates a new org node.
func (h *OrgHandler) CreateNode(c *gin.Context) {
	var req org.CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondBadRequest(c, "invalid request body", err.Error())
		return
	}

	node, err := h.service.CreateNode(req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	// Audit log the creation.
	userID := getCurrentUserID(c)
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: userID,
		ActionType:  audit.ActionCreate,
		TargetType:  "org_node",
		TargetID:    fmt.Sprintf("%d", node.ID),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	})

	c.JSON(http.StatusCreated, gin.H{
		"data": node,
	})
}

// GetNode returns a single org node by ID.
func (h *OrgHandler) GetNode(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	type OrgNodeResponse struct {
		ID         uint   `json:"id"`
		ParentID   *uint  `json:"parent_id"`
		LevelCode  string `json:"level_code"`
		LevelLabel string `json:"level_label"`
		Name       string `json:"name"`
		City       string `json:"city"`
		Department string `json:"department"`
		IsActive   bool   `json:"is_active"`
		SortOrder  int    `json:"sort_order"`
	}

	var resp OrgNodeResponse
	result := h.db.Table("org_nodes").Where("id = ?", id).Scan(&resp)
	if result.Error != nil {
		appErrors.RespondInternalError(c, "failed to load org node")
		return
	}
	if result.RowsAffected == 0 {
		appErrors.RespondNotFound(c, fmt.Sprintf("org node %d not found", id))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": resp,
	})
}

// UpdateNode updates an existing org node.
func (h *OrgHandler) UpdateNode(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	var req org.UpdateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondBadRequest(c, "invalid request body", err.Error())
		return
	}

	node, svcErr := h.service.UpdateNode(id, req)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	// Audit log the update.
	userID := getCurrentUserID(c)
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: userID,
		ActionType:  audit.ActionUpdate,
		TargetType:  "org_node",
		TargetID:    fmt.Sprintf("%d", node.ID),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, gin.H{
		"data": node,
	})
}

// DeleteNode deletes an org node.
func (h *OrgHandler) DeleteNode(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	if svcErr := h.service.DeleteNode(id); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	// Audit log the deletion.
	userID := getCurrentUserID(c)
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: userID,
		ActionType:  audit.ActionDelete,
		TargetType:  "org_node",
		TargetID:    fmt.Sprintf("%d", id),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("org node %d deleted", id),
	})
}

// SwitchContext switches the current user's org context.
func (h *OrgHandler) SwitchContext(c *gin.Context) {
	var req org.SwitchContextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondBadRequest(c, "invalid request body", err.Error())
		return
	}

	userID := getCurrentUserID(c)
	if userID == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}

	if svcErr := h.service.SwitchContext(*userID, req.NodeID); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	// Retrieve the new context for response.
	node, scopeIDs, svcErr := h.service.GetUserContext(*userID)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	// Build breadcrumb.
	var breadcrumb interface{}
	if node != nil {
		bc, err := h.service.GetBreadcrumb(node.ID)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		breadcrumb = bc
	}

	// Audit log the context switch.
	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: userID,
		ActionType:  audit.ActionUpdate,
		TargetType:  "context_assignment",
		TargetID:    fmt.Sprintf("user:%d->node:%d", *userID, req.NodeID),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"current_node": node,
			"scope_ids":    scopeIDs,
			"breadcrumb":   breadcrumb,
		},
	})
}

// GetCurrentContext returns the user's current org context.
func (h *OrgHandler) GetCurrentContext(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID == nil {
		appErrors.RespondUnauthorized(c, "authentication required")
		return
	}

	node, scopeIDs, err := h.service.GetUserContext(*userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	if node == nil {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"current_node": nil,
				"scope_ids":    []uint{},
				"breadcrumb":   []interface{}{},
			},
		})
		return
	}

	breadcrumb, err := h.service.GetBreadcrumb(node.ID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"current_node": node,
			"scope_ids":    scopeIDs,
			"breadcrumb":   breadcrumb,
		},
	})
}

// handleServiceError maps AppError to proper HTTP responses.
func handleServiceError(c *gin.Context, err error) {
	if appErr, ok := err.(*appErrors.AppError); ok {
		switch appErr.Code {
		case "BAD_REQUEST":
			appErrors.RespondWithError(c, http.StatusBadRequest, appErr)
		case "NOT_FOUND":
			appErrors.RespondWithError(c, http.StatusNotFound, appErr)
		case "CONFLICT":
			appErrors.RespondWithError(c, http.StatusConflict, appErr)
		case "VALIDATION_ERROR":
			appErrors.RespondWithError(c, http.StatusUnprocessableEntity, appErr)
		case "FORBIDDEN":
			appErrors.RespondWithError(c, http.StatusForbidden, appErr)
		case "AUTH_REQUIRED":
			appErrors.RespondWithError(c, http.StatusUnauthorized, appErr)
		default:
			appErrors.RespondWithError(c, http.StatusInternalServerError, appErr)
		}
		return
	}
	appErrors.RespondInternalError(c, err.Error())
}

// parseIDParam parses a uint route parameter.
func parseIDParam(c *gin.Context, param string) (uint, error) {
	idStr := c.Param(param)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, fmt.Sprintf("invalid %s parameter: %s", param, idStr), nil)
		return 0, err
	}
	return uint(id), nil
}

// getCurrentUserID retrieves the current authenticated user's ID from the Gin context.
func getCurrentUserID(c *gin.Context) *uint {
	user := auth.GetCurrentUser(c)
	if user != nil {
		return &user.ID
	}
	return nil
}
