package reports

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"backend/internal/logging"
	"backend/internal/models"
)

// ReportService provides report scheduling, generation, and download.
type ReportService struct {
	db *gorm.DB
}

// NewReportService creates a new ReportService instance.
func NewReportService(db *gorm.DB) *ReportService {
	return &ReportService{db: db}
}

// CreateScheduleRequest is the request body for creating a report schedule.
type CreateScheduleRequest struct {
	Name         string `json:"name" binding:"required"`
	KPICode      string `json:"kpi_code"`
	CronExpr     string `json:"cron_expr" binding:"required"`
	Timezone     string `json:"timezone" binding:"required"`
	OutputFormat string `json:"output_format" binding:"required"`
	ScopeJSON    string `json:"scope_json"`
	Recipients   string `json:"recipients"`
	Enabled      bool   `json:"enabled"`
}

// UpdateScheduleRequest is the request body for updating a report schedule.
type UpdateScheduleRequest struct {
	Name         *string `json:"name"`
	KPICode      *string `json:"kpi_code"`
	CronExpr     *string `json:"cron_expr"`
	Timezone     *string `json:"timezone"`
	OutputFormat *string `json:"output_format"`
	ScopeJSON    *string `json:"scope_json"`
	Recipients   *string `json:"recipients"`
	Enabled      *bool   `json:"enabled"`
}

// ScheduleFilter holds the parameters for filtering report schedules.
type ScheduleFilter struct {
	IsActive *bool
	Page     int
	PageSize int
}

// RunFilter holds the parameters for filtering report runs.
type RunFilter struct {
	ScheduleID uint
	State      string
	DateFrom   string
	DateTo     string
	Page       int
	PageSize   int
	UserCity   string // scope filter: restrict to schedules matching user city
	UserDept   string // scope filter: restrict to schedules matching user dept
}

// CreateSchedule creates a new report schedule.
func (s *ReportService) CreateSchedule(req CreateScheduleRequest, createdBy uint) (*models.ReportSchedule, error) {
	if req.OutputFormat != "csv" && req.OutputFormat != "pdf" && req.OutputFormat != "xlsx" {
		return nil, fmt.Errorf("output_format must be csv, pdf, or xlsx")
	}

	schedule := models.ReportSchedule{
		Name:         req.Name,
		KPICode:      req.KPICode,
		CronExpr:     req.CronExpr,
		Timezone:     req.Timezone,
		OutputFormat: req.OutputFormat,
		ScopeJSON:    req.ScopeJSON,
		Recipients:   req.Recipients,
		IsActive:     req.Enabled,
		CreatedBy:    createdBy,
	}

	if err := s.db.Create(&schedule).Error; err != nil {
		return nil, fmt.Errorf("failed to create report schedule: %w", err)
	}

	return &schedule, nil
}

// UpdateSchedule updates an existing report schedule.
func (s *ReportService) UpdateSchedule(id uint, req UpdateScheduleRequest) (*models.ReportSchedule, error) {
	var schedule models.ReportSchedule
	if err := s.db.First(&schedule, id).Error; err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.KPICode != nil {
		updates["kpi_code"] = *req.KPICode
	}
	if req.CronExpr != nil {
		updates["cron_expr"] = *req.CronExpr
	}
	if req.Timezone != nil {
		updates["timezone"] = *req.Timezone
	}
	if req.OutputFormat != nil {
		if *req.OutputFormat != "csv" && *req.OutputFormat != "pdf" && *req.OutputFormat != "xlsx" {
			return nil, fmt.Errorf("output_format must be csv, pdf, or xlsx")
		}
		updates["output_format"] = *req.OutputFormat
	}
	if req.ScopeJSON != nil {
		updates["scope_json"] = *req.ScopeJSON
	}
	if req.Recipients != nil {
		updates["recipients"] = *req.Recipients
	}
	if req.Enabled != nil {
		updates["is_active"] = *req.Enabled
	}

	if len(updates) > 0 {
		if err := s.db.Model(&schedule).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("failed to update report schedule: %w", err)
		}
	}

	// Re-fetch to return current state
	s.db.First(&schedule, id)
	return &schedule, nil
}

// DeleteSchedule soft-deletes a report schedule by deactivating it.
func (s *ReportService) DeleteSchedule(id uint) error {
	result := s.db.Model(&models.ReportSchedule{}).Where("id = ?", id).Update("is_active", false)
	if result.Error != nil {
		return fmt.Errorf("failed to deactivate schedule: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetSchedule returns a single schedule by ID.
func (s *ReportService) GetSchedule(id uint) (*models.ReportSchedule, error) {
	var schedule models.ReportSchedule
	if err := s.db.First(&schedule, id).Error; err != nil {
		return nil, err
	}
	return &schedule, nil
}

// GetSchedules returns all schedules with pagination and optional active filter.
func (s *ReportService) GetSchedules(filter ScheduleFilter) ([]models.ReportSchedule, int64, error) {
	query := s.db.Model(&models.ReportSchedule{})

	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count schedules: %w", err)
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
	var schedules []models.ReportSchedule
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&schedules).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve schedules: %w", err)
	}

	return schedules, total, nil
}

// GetRun returns a single report run by ID.
func (s *ReportService) GetRun(id uint) (*models.ReportRun, error) {
	var run models.ReportRun
	if err := s.db.First(&run, id).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

// GetRuns returns report runs with pagination and filtering.
func (s *ReportService) GetRuns(filter RunFilter) ([]models.ReportRun, int64, error) {
	query := s.db.Model(&models.ReportRun{})

	if filter.ScheduleID > 0 {
		query = query.Where("schedule_id = ?", filter.ScheduleID)
	}
	if filter.State != "" {
		query = query.Where("state = ?", filter.State)
	}
	if filter.DateFrom != "" {
		if t, err := time.Parse("2006-01-02", filter.DateFrom); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if filter.DateTo != "" {
		if t, err := time.Parse("2006-01-02", filter.DateTo); err == nil {
			query = query.Where("created_at <= ?", t.Add(24*time.Hour))
		}
	}

	// Scope filtering: restrict runs to schedules matching user's city/dept.
	if filter.UserCity != "" && filter.UserCity != "*" {
		query = query.Where("schedule_id IN (SELECT id FROM report_schedules WHERE scope_json IS NULL OR scope_json->'$.city' = ? OR scope_json->'$.city' IS NULL)", filter.UserCity)
	}
	if filter.UserDept != "" && filter.UserDept != "*" {
		query = query.Where("schedule_id IN (SELECT id FROM report_schedules WHERE scope_json IS NULL OR scope_json->'$.department' = ? OR scope_json->'$.department' IS NULL)", filter.UserDept)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count report runs: %w", err)
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
	var runs []models.ReportRun
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&runs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve report runs: %w", err)
	}

	return runs, total, nil
}

// ExecuteReport generates a report for a given schedule.
// It creates a ReportRun record, generates the output file, and updates the run state.
func (s *ReportService) ExecuteReport(scheduleID uint, requestedBy uint) (*models.ReportRun, error) {
	var schedule models.ReportSchedule
	if err := s.db.First(&schedule, scheduleID).Error; err != nil {
		return nil, fmt.Errorf("schedule not found: %w", err)
	}

	now := time.Now()
	run := models.ReportRun{
		ScheduleID:  scheduleID,
		State:       "running",
		RequestedBy: &requestedBy,
		StartedAt:   &now,
	}
	if err := s.db.Create(&run).Error; err != nil {
		return nil, fmt.Errorf("failed to create report run: %w", err)
	}

	// Generate the report
	outputPath, rowCount, err := s.generateReport(&schedule, &run)
	if err != nil {
		failReason := err.Error()
		completedAt := time.Now()
		s.db.Model(&run).Updates(map[string]interface{}{
			"state":          "failed",
			"failure_reason": failReason,
			"completed_at":   completedAt,
		})
		run.State = "failed"
		run.FailureReason = failReason
		run.CompletedAt = &completedAt
		logging.Error("reports", "execute_report",
			fmt.Sprintf("report generation failed for schedule %d: %v", scheduleID, err))
		return &run, nil
	}

	completedAt := time.Now()
	s.db.Model(&run).Updates(map[string]interface{}{
		"state":        "ready",
		"output_path":  outputPath,
		"row_count":    rowCount,
		"completed_at": completedAt,
	})
	run.State = "ready"
	run.OutputPath = outputPath
	run.RowCount = rowCount
	run.CompletedAt = &completedAt

	return &run, nil
}

// generateReport creates the output file for a report schedule.
func (s *ReportService) generateReport(schedule *models.ReportSchedule, run *models.ReportRun) (string, int, error) {
	exportDir := filepath.Join("exports", "reports")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create export directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	sanitizedName := strings.ReplaceAll(schedule.Name, " ", "_")
	filename := fmt.Sprintf("%s_%d_%s", sanitizedName, run.ID, timestamp)

	switch schedule.OutputFormat {
	case "csv":
		return s.generateCSVReport(schedule, exportDir, filename)
	case "pdf":
		return s.generatePDFReport(schedule, exportDir, filename)
	case "xlsx":
		return s.generateXLSXReport(schedule, exportDir, filename)
	default:
		return s.generateCSVReport(schedule, exportDir, filename)
	}
}

// generateCSVReport generates a CSV report file.
func (s *ReportService) generateCSVReport(schedule *models.ReportSchedule, dir, filename string) (string, int, error) {
	outputPath := filepath.Join(dir, filename+".csv")

	records, err := s.queryReportRecords(schedule)
	if err != nil {
		return "", 0, err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"id", "entity_type", "natural_key", "status", "created_at", "updated_at"}
	if err := writer.Write(header); err != nil {
		return "", 0, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write rows
	for _, r := range records {
		row := []string{
			fmt.Sprintf("%d", r.ID),
			r.EntityType,
			r.NaturalKey,
			r.Status,
			r.CreatedAt.Format(time.RFC3339),
			r.UpdatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return "", 0, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return outputPath, len(records), nil
}

// generatePDFReport generates a PDF report file.
func (s *ReportService) generatePDFReport(schedule *models.ReportSchedule, dir, filename string) (string, int, error) {
	outputPath := filepath.Join(dir, filename+".pdf")

	records, err := s.queryReportRecords(schedule)
	if err != nil {
		return "", 0, err
	}

	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetFont("Arial", "B", 14)
	pdf.AddPage()

	// Title
	pdf.CellFormat(0, 10, schedule.Name, "", 1, "C", false, 0, "")
	pdf.Ln(4)
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(0, 6, fmt.Sprintf("Generated: %s", time.Now().Format(time.RFC3339)), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	// Table header
	headers := []string{"ID", "Entity Type", "Natural Key", "Status", "Created At", "Updated At"}
	colWidths := []float64{20, 50, 60, 30, 60, 60}
	pdf.SetFont("Arial", "B", 9)
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 8, h, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	// Table rows
	pdf.SetFont("Arial", "", 8)
	for _, r := range records {
		pdf.CellFormat(colWidths[0], 7, fmt.Sprintf("%d", r.ID), "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[1], 7, r.EntityType, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[2], 7, r.NaturalKey, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[3], 7, r.Status, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[4], 7, r.CreatedAt.Format(time.RFC3339), "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[5], 7, r.UpdatedAt.Format(time.RFC3339), "1", 0, "L", false, 0, "")
		pdf.Ln(-1)
	}

	if err := pdf.OutputFileAndClose(outputPath); err != nil {
		return "", 0, fmt.Errorf("failed to write PDF: %w", err)
	}

	return outputPath, len(records), nil
}

// generateXLSXReport generates an XLSX report file.
func (s *ReportService) generateXLSXReport(schedule *models.ReportSchedule, dir, filename string) (string, int, error) {
	outputPath := filepath.Join(dir, filename+".xlsx")

	records, err := s.queryReportRecords(schedule)
	if err != nil {
		return "", 0, err
	}

	f := excelize.NewFile()
	sheet := "Report"
	idx, _ := f.NewSheet(sheet)
	f.SetActiveSheet(idx)

	// Write header
	headers := []string{"ID", "Entity Type", "Natural Key", "Status", "Created At", "Updated At"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Write rows
	for rowIdx, r := range records {
		row := rowIdx + 2
		f.SetCellValue(sheet, cellName(1, row), r.ID)
		f.SetCellValue(sheet, cellName(2, row), r.EntityType)
		f.SetCellValue(sheet, cellName(3, row), r.NaturalKey)
		f.SetCellValue(sheet, cellName(4, row), r.Status)
		f.SetCellValue(sheet, cellName(5, row), r.CreatedAt.Format(time.RFC3339))
		f.SetCellValue(sheet, cellName(6, row), r.UpdatedAt.Format(time.RFC3339))
	}

	// Delete default "Sheet1" if different
	if sheet != "Sheet1" {
		f.DeleteSheet("Sheet1")
	}

	if err := f.SaveAs(outputPath); err != nil {
		return "", 0, fmt.Errorf("failed to write XLSX: %w", err)
	}

	return outputPath, len(records), nil
}

// queryReportRecords fetches the master records used by report generation.
func (s *ReportService) queryReportRecords(schedule *models.ReportSchedule) ([]models.MasterRecord, error) {
	var scopeFilter struct {
		EntityType string `json:"entity_type"`
		City       string `json:"city"`
		Department string `json:"department"`
	}
	if schedule.ScopeJSON != "" {
		_ = json.Unmarshal([]byte(schedule.ScopeJSON), &scopeFilter)
	}

	query := s.db.Model(&models.MasterRecord{})
	if scopeFilter.EntityType != "" {
		query = query.Where("entity_type = ?", scopeFilter.EntityType)
	}

	// Apply city/department scope filtering from the schedule's scope_json.
	if scopeFilter.City != "" {
		// Filter records linked to versions whose scope_key maps to org nodes
		// in the target city.
		query = query.Where(
			"id IN (SELECT mvi.master_record_id FROM master_version_items mvi "+
				"JOIN master_versions mv ON mv.id = mvi.version_id "+
				"WHERE mv.scope_key IN (SELECT CONCAT('node:', id) FROM org_nodes WHERE city = ?))",
			scopeFilter.City)
	}
	if scopeFilter.Department != "" {
		query = query.Where(
			"id IN (SELECT mvi.master_record_id FROM master_version_items mvi "+
				"JOIN master_versions mv ON mv.id = mvi.version_id "+
				"WHERE mv.scope_key IN (SELECT CONCAT('node:', id) FROM org_nodes WHERE department = ?))",
			scopeFilter.Department)
	}

	var records []models.MasterRecord
	if err := query.Order("id ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}
	return records, nil
}

// cellName is a helper to convert column/row to Excel cell name.
func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

// DownloadReport returns the file path for a completed report run.
func (s *ReportService) DownloadReport(runID uint) (string, error) {
	var run models.ReportRun
	if err := s.db.First(&run, runID).Error; err != nil {
		return "", err
	}

	if run.State != "ready" {
		return "", fmt.Errorf("report is not ready for download (state: %s)", run.State)
	}

	if run.OutputPath == "" {
		return "", fmt.Errorf("report output path is empty")
	}

	// Verify file exists
	if _, err := os.Stat(run.OutputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("report file not found on disk")
	}

	return run.OutputPath, nil
}

// CheckAccess re-checks whether a user still has permission to download a report at download time.
func (s *ReportService) CheckAccess(runID uint, userID uint, userRole string, userCity string, userDept string) error {
	var run models.ReportRun
	if err := s.db.First(&run, runID).Error; err != nil {
		return fmt.Errorf("report run not found")
	}

	// System admins always have access
	if userRole == "system_admin" {
		return nil
	}

	// Load the schedule to check scope
	var schedule models.ReportSchedule
	if err := s.db.First(&schedule, run.ScheduleID).Error; err != nil {
		return fmt.Errorf("schedule not found")
	}

	// Deny by default if user has no scope assigned (empty scope = no access, not full access)
	if userCity == "" && userDept == "" {
		return fmt.Errorf("access denied: user has no scope assigned")
	}

	// Check scope
	if schedule.ScopeJSON != "" {
		var scope struct {
			City       string `json:"city"`
			Department string `json:"department"`
		}
		if err := json.Unmarshal([]byte(schedule.ScopeJSON), &scope); err == nil {
			// If report targets a city, user must have matching or wildcard city scope
			if scope.City != "" {
				if userCity == "" {
					return fmt.Errorf("access denied: user has no city scope for this report")
				}
				if userCity != "*" && scope.City != userCity {
					return fmt.Errorf("access denied: report scope is outside your city")
				}
			}
			// If report targets a department, user must have matching or wildcard dept scope
			if scope.Department != "" {
				if userDept == "" {
					return fmt.Errorf("access denied: user has no department scope for this report")
				}
				if userDept != "*" && scope.Department != userDept {
					return fmt.Errorf("access denied: report scope is outside your department")
				}
			}
		}
	}

	return nil
}
