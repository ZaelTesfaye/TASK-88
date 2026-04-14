package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/internal/audit"
	appErrors "backend/internal/errors"
	"backend/internal/rbac"
	"backend/internal/reports"
)

type ReportHandler struct {
	db      *gorm.DB
	service *reports.ReportService
}

func NewReportHandler(db *gorm.DB) *ReportHandler {
	return &ReportHandler{
		db:      db,
		service: reports.NewReportService(db),
	}
}

func (h *ReportHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rpts := rg.Group("/reports")
	{
		rpts.GET("/schedules", h.ListSchedules)
		rpts.POST("/schedules", h.CreateSchedule)
		rpts.GET("/schedules/:id", h.GetSchedule)
		rpts.PATCH("/schedules/:id", h.UpdateSchedule)
		rpts.DELETE("/schedules/:id", h.DeleteSchedule)
		rpts.POST("/schedules/:id/trigger", h.TriggerSchedule)

		rpts.GET("/runs", h.ListRuns)
		rpts.GET("/runs/:id", h.GetRun)
		rpts.GET("/runs/:id/download", h.DownloadRun)
		rpts.GET("/runs/:id/access-check", h.AccessCheck)
	}
}

func (h *ReportHandler) ListSchedules(c *gin.Context) {
	var filter reports.ScheduleFilter
	if p := c.Query("page"); p != "" {
		filter.Page, _ = strconv.Atoi(p)
	}
	if ps := c.Query("page_size"); ps != "" {
		filter.PageSize, _ = strconv.Atoi(ps)
	}
	if active := c.Query("active"); active != "" {
		val := active == "true"
		filter.IsActive = &val
	}

	schedules, total, err := h.service.GetSchedules(filter)
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": schedules, "total": total})
}

func (h *ReportHandler) CreateSchedule(c *gin.Context) {
	var req reports.CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}

	var userID uint
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(uint)
	}

	schedule, err := h.service.CreateSchedule(req, userID)
	if err != nil {
		appErrors.RespondBadRequest(c, err.Error(), nil)
		return
	}

	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: &userID,
		ActionType:  audit.ActionCreate,
		TargetType:  "report_schedule",
		TargetID:    fmt.Sprintf("%d", schedule.ID),
	})

	c.JSON(http.StatusCreated, schedule)
}

func (h *ReportHandler) GetSchedule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid schedule ID", nil)
		return
	}

	schedule, err := h.service.GetSchedule(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "schedule not found")
			return
		}
		appErrors.RespondInternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, schedule)
}

func (h *ReportHandler) UpdateSchedule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid schedule ID", nil)
		return
	}

	var req reports.UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.RespondValidationError(c, "invalid request body", err.Error())
		return
	}

	schedule, err := h.service.UpdateSchedule(uint(id), req)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "schedule not found")
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
			TargetType:  "report_schedule",
			TargetID:    fmt.Sprintf("%d", id),
		})
	}

	c.JSON(http.StatusOK, schedule)
}

func (h *ReportHandler) DeleteSchedule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid schedule ID", nil)
		return
	}

	if err := h.service.DeleteSchedule(uint(id)); err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "schedule not found")
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
			TargetType:  "report_schedule",
			TargetID:    fmt.Sprintf("%d", id),
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "schedule deactivated"})
}

func (h *ReportHandler) TriggerSchedule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid schedule ID", nil)
		return
	}

	var userID uint
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(uint)
	}

	run, err := h.service.ExecuteReport(uint(id), userID)
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}

	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: &userID,
		ActionType:  audit.ActionCreate,
		TargetType:  "report_run",
		TargetID:    fmt.Sprintf("%d", run.ID),
	})

	c.JSON(http.StatusCreated, run)
}

func (h *ReportHandler) ListRuns(c *gin.Context) {
	var filter reports.RunFilter
	if sid := c.Query("schedule_id"); sid != "" {
		v, _ := strconv.ParseUint(sid, 10, 32)
		filter.ScheduleID = uint(v)
	}
	filter.State = c.Query("state")
	filter.DateFrom = c.Query("date_from")
	filter.DateTo = c.Query("date_to")
	if p := c.Query("page"); p != "" {
		filter.Page, _ = strconv.Atoi(p)
	}
	if ps := c.Query("page_size"); ps != "" {
		filter.PageSize, _ = strconv.Atoi(ps)
	}

	// Apply requester-scope filtering: restrict to schedules the user can see.
	userRole, _ := c.Get("user_role")
	userCity := c.GetString("city_scope")
	userDept := c.GetString("dept_scope")
	if userRole != rbac.SystemAdmin {
		filter.UserCity = userCity
		filter.UserDept = userDept
	}

	runs, total, err := h.service.GetRuns(filter)
	if err != nil {
		appErrors.RespondInternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": runs, "total": total})
}

func (h *ReportHandler) GetRun(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid run ID", nil)
		return
	}

	run, err := h.service.GetRun(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			appErrors.RespondNotFound(c, "report run not found")
			return
		}
		appErrors.RespondInternalError(c, err.Error())
		return
	}

	// Per-run scope check: non-admins must have matching scope.
	var userID uint
	var userRole, userCity, userDept string
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(uint)
	}
	if r, exists := c.Get("user_role"); exists {
		userRole = r.(string)
	}
	if cs, exists := c.Get("city_scope"); exists {
		userCity = fmt.Sprint(cs)
	}
	if ds, exists := c.Get("dept_scope"); exists {
		userDept = fmt.Sprint(ds)
	}

	if accessErr := h.service.CheckAccess(uint(id), userID, userRole, userCity, userDept); accessErr != nil {
		appErrors.RespondForbidden(c, accessErr.Error())
		return
	}

	c.JSON(http.StatusOK, run)
}

func (h *ReportHandler) DownloadRun(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid run ID", nil)
		return
	}

	var userID uint
	var userRole, userCity, userDept string
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(uint)
	}
	if r, exists := c.Get("user_role"); exists {
		userRole = r.(string)
	}
	if cs, exists := c.Get("city_scope"); exists {
		userCity = fmt.Sprint(cs)
	}
	if ds, exists := c.Get("dept_scope"); exists {
		userDept = fmt.Sprint(ds)
	}

	if accessErr := h.service.CheckAccess(uint(id), userID, userRole, userCity, userDept); accessErr != nil {
		appErrors.RespondForbidden(c, accessErr.Error())
		return
	}

	filePath, err := h.service.DownloadReport(uint(id))
	if err != nil {
		appErrors.RespondBadRequest(c, err.Error(), nil)
		return
	}

	_ = audit.LogAction(h.db, audit.AuditEntry{
		ActorUserID: &userID,
		ActionType:  audit.ActionExport,
		TargetType:  "report_run",
		TargetID:    fmt.Sprintf("%d", id),
	})

	c.File(filePath)
}

func (h *ReportHandler) AccessCheck(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appErrors.RespondBadRequest(c, "invalid run ID", nil)
		return
	}

	var userID uint
	var userRole, userCity, userDept string
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(uint)
	}
	if r, exists := c.Get("user_role"); exists {
		userRole = r.(string)
	}
	if cs, exists := c.Get("city_scope"); exists {
		userCity = fmt.Sprint(cs)
	}
	if ds, exists := c.Get("dept_scope"); exists {
		userDept = fmt.Sprint(ds)
	}

	if accessErr := h.service.CheckAccess(uint(id), userID, userRole, userCity, userDept); accessErr != nil {
		c.JSON(http.StatusOK, gin.H{"has_access": false, "reason": accessErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"has_access": true})
}
