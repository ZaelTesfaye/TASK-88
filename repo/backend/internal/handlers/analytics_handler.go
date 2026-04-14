package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/internal/analytics"
	"backend/internal/audit"
	appErrors "backend/internal/errors"
	"backend/internal/models"
)

type AnalyticsHandler struct {
	db      *gorm.DB
	service *analytics.AnalyticsService
}

func NewAnalyticsHandler(db *gorm.DB) *AnalyticsHandler {
	return &AnalyticsHandler{
		db:      db,
		service: analytics.NewAnalyticsService(db),
	}
}

func (h *AnalyticsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	ag := rg.Group("/analytics")
	{
		ag.GET("/kpis", h.GetKPIs)
		ag.GET("/kpis/definitions", h.ListKPIDefinitions)
		ag.POST("/kpis/definitions", h.CreateKPIDefinition)
		ag.GET("/kpis/definitions/:code", h.GetKPIDefinition)
		ag.PUT("/kpis/definitions/:code", h.UpdateKPIDefinition)
		ag.DELETE("/kpis/definitions/:code", h.DeleteKPIDefinition)
		ag.GET("/trends", h.GetTrends)
	}
}

func (h *AnalyticsHandler) GetKPIs(c *gin.Context) {
	filter := analytics.KPIFilter{
		CityScope: c.GetString("city_scope"),
		DeptScope: c.GetString("dept_scope"),
		DateFrom:  c.Query("date_from"),
		DateTo:    c.Query("date_to"),
	}

	results, err := h.service.GetKPIs(filter)
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"kpis": results})
}

func (h *AnalyticsHandler) ListKPIDefinitions(c *gin.Context) {
	defs, err := h.service.GetKPIDefinitions()
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": defs, "total": len(defs)})
}

func (h *AnalyticsHandler) GetKPIDefinition(c *gin.Context) {
	code := c.Param("code")
	def, err := h.service.GetKPIDefinitionByCode(code)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "KPI definition not found")
			return
		}
		appErrors.RespondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, def)
}

func (h *AnalyticsHandler) CreateKPIDefinition(c *gin.Context) {
	var def models.AnalyticsKPIDefinition
	if err := c.ShouldBindJSON(&def); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}

	if err := h.service.CreateKPIDefinition(&def); err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}

	if uid, exists := c.Get("user_id"); exists {
		userID := uid.(uint)
		_ = audit.LogAction(h.db, audit.AuditEntry{
			ActorUserID: &userID,
			ActionType:  audit.ActionCreate,
			TargetType:  "kpi_definition",
			TargetID:    def.Code,
		})
	}

	c.JSON(http.StatusCreated, def)
}

func (h *AnalyticsHandler) UpdateKPIDefinition(c *gin.Context) {
	code := c.Param("code")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}

	def, err := h.service.UpdateKPIDefinition(code, updates)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "KPI definition not found")
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
			TargetType:  "kpi_definition",
			TargetID:    code,
		})
	}

	c.JSON(http.StatusOK, def)
}

func (h *AnalyticsHandler) DeleteKPIDefinition(c *gin.Context) {
	code := c.Param("code")

	if err := h.service.DeleteKPIDefinition(code); err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "KPI definition not found")
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
			TargetType:  "kpi_definition",
			TargetID:    code,
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "KPI definition deactivated"})
}

func (h *AnalyticsHandler) GetTrends(c *gin.Context) {
	filter := analytics.TrendFilter{
		CityScope:   c.GetString("city_scope"),
		DeptScope:   c.GetString("dept_scope"),
		DateFrom:    c.Query("date_from"),
		DateTo:      c.Query("date_to"),
		Granularity: c.Query("granularity"),
	}

	if codes := c.Query("kpi_codes"); codes != "" {
		filter.KPICodes = strings.Split(codes, ",")
	}

	series, err := h.service.GetTrends(filter)
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"series": series})
}
