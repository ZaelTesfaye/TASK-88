package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/internal/audit"
	appErrors "backend/internal/errors"
	"backend/internal/integration"
	"backend/internal/models"
)

type IntegrationHandler struct {
	db      *gorm.DB
	service *integration.IntegrationService
}

func NewIntegrationHandler(db *gorm.DB) *IntegrationHandler {
	return &IntegrationHandler{
		db:      db,
		service: integration.NewIntegrationService(db),
	}
}

func (h *IntegrationHandler) RegisterRoutes(rg *gin.RouterGroup) {
	ig := rg.Group("/integrations")
	{
		ig.GET("/endpoints", h.ListEndpoints)
		ig.POST("/endpoints", h.CreateEndpoint)
		ig.GET("/endpoints/:id", h.GetEndpoint)
		ig.PUT("/endpoints/:id", h.UpdateEndpoint)
		ig.DELETE("/endpoints/:id", h.DeleteEndpoint)
		ig.POST("/endpoints/:id/test", h.TestEndpoint)

		ig.GET("/deliveries", h.ListDeliveries)
		ig.GET("/deliveries/:id", h.GetDelivery)
		ig.POST("/deliveries/:id/retry", h.RetryDelivery)

		ig.GET("/connectors", h.ListConnectors)
		ig.POST("/connectors", h.CreateConnector)
		ig.GET("/connectors/:id", h.GetConnector)
		ig.PUT("/connectors/:id", h.UpdateConnector)
		ig.DELETE("/connectors/:id", h.DeleteConnector)
		ig.POST("/connectors/:id/health-check", h.HealthCheckConnector)
	}
}

func (h *IntegrationHandler) ListEndpoints(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	endpoints, total, err := h.service.ListEndpoints(page, pageSize)
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": endpoints, "total": total})
}

func (h *IntegrationHandler) CreateEndpoint(c *gin.Context) {
	var req integration.CreateEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}
	endpoint, err := h.service.CreateEndpoint(req)
	if err != nil {
		appErrors.RespondBadRequest(c, err.Error(), nil)
		return
	}
	if uid, exists := c.Get("user_id"); exists {
		userID := uid.(uint)
		_ = audit.LogAction(h.db, audit.AuditEntry{
			ActorUserID: &userID,
			ActionType:  audit.ActionCreate,
			TargetType:  "integration_endpoint",
			TargetID:    fmt.Sprintf("%d", endpoint.ID),
		})
	}
	c.JSON(http.StatusCreated, endpoint)
}

func (h *IntegrationHandler) GetEndpoint(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid endpoint ID", nil)
		return
	}
	endpoint, err := h.service.GetEndpoint(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "endpoint not found")
			return
		}
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, endpoint)
}

func (h *IntegrationHandler) UpdateEndpoint(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid endpoint ID", nil)
		return
	}
	var req integration.UpdateEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}
	endpoint, err := h.service.UpdateEndpoint(uint(id), req)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "endpoint not found")
			return
		}
		appErrors.RespondBadRequest(c, err.Error(), nil)
		return
	}
	if uid, exists := c.Get("user_id"); exists {
		userID := uid.(uint)
		_ = audit.LogAction(h.db, audit.AuditEntry{
			ActorUserID: &userID,
			ActionType:  audit.ActionUpdate,
			TargetType:  "integration_endpoint",
			TargetID:    fmt.Sprintf("%d", id),
		})
	}
	c.JSON(http.StatusOK, endpoint)
}

func (h *IntegrationHandler) DeleteEndpoint(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid endpoint ID", nil)
		return
	}
	if err := h.service.DeleteEndpoint(uint(id)); err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "endpoint not found")
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
			TargetType:  "integration_endpoint",
			TargetID:    fmt.Sprintf("%d", id),
		})
	}
	c.JSON(http.StatusOK, gin.H{"message": "endpoint deleted"})
}

func (h *IntegrationHandler) TestEndpoint(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid endpoint ID", nil)
		return
	}
	statusCode, body, err := h.service.TestEndpoint(uint(id))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": statusCode >= 200 && statusCode < 300, "status_code": statusCode, "body": body})
}

func (h *IntegrationHandler) ListDeliveries(c *gin.Context) {
	var filter integration.DeliveryFilter
	if eid := c.Query("endpoint_id"); eid != "" {
		v, _ := strconv.ParseUint(eid, 10, 32)
		filter.EndpointID = uint(v)
	}
	filter.EventID = c.Query("event_id")
	filter.State = c.Query("state")
	if p := c.Query("page"); p != "" {
		filter.Page, _ = strconv.Atoi(p)
	}
	if ps := c.Query("page_size"); ps != "" {
		filter.PageSize, _ = strconv.Atoi(ps)
	}

	deliveries, total, err := h.service.GetDeliveries(filter)
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": deliveries, "total": total})
}

func (h *IntegrationHandler) GetDelivery(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid delivery ID", nil)
		return
	}
	delivery, err := h.service.GetDelivery(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "delivery not found")
			return
		}
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, delivery)
}

func (h *IntegrationHandler) RetryDelivery(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid delivery ID", nil)
		return
	}
	if err := h.service.RetryDelivery(uint(id)); err != nil {
		appErrors.RespondBadRequest(c, err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "delivery retry initiated"})
}

func (h *IntegrationHandler) ListConnectors(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	connectors, total, err := h.service.ListConnectors(page, pageSize)
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": connectors, "total": total})
}

func (h *IntegrationHandler) CreateConnector(c *gin.Context) {
	var conn models.ConnectorDefinition
	if err := c.ShouldBindJSON(&conn); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}
	if err := h.service.CreateConnector(&conn); err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	if uid, exists := c.Get("user_id"); exists {
		userID := uid.(uint)
		_ = audit.LogAction(h.db, audit.AuditEntry{
			ActorUserID: &userID,
			ActionType:  audit.ActionCreate,
			TargetType:  "connector_definition",
			TargetID:    fmt.Sprintf("%d", conn.ID),
		})
	}
	c.JSON(http.StatusCreated, conn)
}

func (h *IntegrationHandler) GetConnector(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid connector ID", nil)
		return
	}
	conn, err := h.service.GetConnector(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "connector not found")
			return
		}
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, conn)
}

func (h *IntegrationHandler) UpdateConnector(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid connector ID", nil)
		return
	}
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}
	conn, err := h.service.UpdateConnector(uint(id), updates)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "connector not found")
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
			TargetType:  "connector_definition",
			TargetID:    fmt.Sprintf("%d", id),
		})
	}
	c.JSON(http.StatusOK, conn)
}

func (h *IntegrationHandler) DeleteConnector(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid connector ID", nil)
		return
	}
	if err := h.service.DeleteConnector(uint(id)); err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "connector not found")
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
			TargetType:  "connector_definition",
			TargetID:    fmt.Sprintf("%d", id),
		})
	}
	c.JSON(http.StatusOK, gin.H{"message": "connector deleted"})
}

func (h *IntegrationHandler) HealthCheckConnector(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid connector ID", nil)
		return
	}
	conn, err := h.service.HealthCheckConnector(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "connector not found")
			return
		}
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"connector_id": conn.ID, "health_status": conn.HealthStatus, "last_check": conn.LastHealthCheckAt})
}
