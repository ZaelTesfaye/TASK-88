package analytics

import (
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"

	"backend/internal/logging"
	"backend/internal/models"
)

// AnalyticsService provides KPI evaluation and trend analysis.
type AnalyticsService struct {
	db *gorm.DB
}

// NewAnalyticsService creates a new AnalyticsService instance.
func NewAnalyticsService(db *gorm.DB) *AnalyticsService {
	return &AnalyticsService{db: db}
}

// KPIFilter holds the parameters for filtering KPI results.
type KPIFilter struct {
	NodeIDs   []uint
	CityScope string
	DeptScope string
	DateFrom  string
	DateTo    string
}

// KPIResult holds the computed value of a single KPI.
type KPIResult struct {
	Code            string  `json:"code"`
	Label           string  `json:"label"`
	Value           float64 `json:"value"`
	PrevValue       float64 `json:"prev_value"`
	ChangePercent   float64 `json:"change_percent"`
	TrendDirection  string  `json:"trend_direction"`
	Unit            string  `json:"unit"`
}

// TrendFilter holds the parameters for filtering trend data.
type TrendFilter struct {
	KPICodes    []string
	NodeIDs     []uint
	CityScope   string
	DeptScope   string
	DateFrom    string
	DateTo      string
	Granularity string // daily, weekly, monthly
}

// TrendSeries represents a time-series for a single KPI.
type TrendSeries struct {
	Code   string      `json:"code"`
	Label  string      `json:"label"`
	Points []DataPoint `json:"points"`
}

// DataPoint is a single point in a trend time-series.
type DataPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// GetKPIDefinitions returns all active KPI definitions.
func (s *AnalyticsService) GetKPIDefinitions() ([]models.AnalyticsKPIDefinition, error) {
	var defs []models.AnalyticsKPIDefinition
	if err := s.db.Where("is_active = ?", true).Order("code ASC").Find(&defs).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve KPI definitions: %w", err)
	}
	return defs, nil
}

// GetKPIDefinitionByCode returns a single KPI definition by its code.
func (s *AnalyticsService) GetKPIDefinitionByCode(code string) (*models.AnalyticsKPIDefinition, error) {
	var def models.AnalyticsKPIDefinition
	if err := s.db.Where("code = ?", code).First(&def).Error; err != nil {
		return nil, err
	}
	return &def, nil
}

// CreateKPIDefinition creates a new KPI definition.
func (s *AnalyticsService) CreateKPIDefinition(def *models.AnalyticsKPIDefinition) error {
	if err := s.db.Create(def).Error; err != nil {
		return fmt.Errorf("failed to create KPI definition: %w", err)
	}
	return nil
}

// UpdateKPIDefinition updates an existing KPI definition identified by code.
func (s *AnalyticsService) UpdateKPIDefinition(code string, updates map[string]interface{}) (*models.AnalyticsKPIDefinition, error) {
	var def models.AnalyticsKPIDefinition
	if err := s.db.Where("code = ?", code).First(&def).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&def).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update KPI definition: %w", err)
	}
	return &def, nil
}

// DeleteKPIDefinition soft-deletes a KPI definition by marking it inactive.
func (s *AnalyticsService) DeleteKPIDefinition(code string) error {
	result := s.db.Model(&models.AnalyticsKPIDefinition{}).Where("code = ?", code).Update("is_active", false)
	if result.Error != nil {
		return fmt.Errorf("failed to deactivate KPI definition: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetKPIs computes all active KPIs applying scope filters and returns current + previous values.
func (s *AnalyticsService) GetKPIs(filter KPIFilter) ([]KPIResult, error) {
	defs, err := s.GetKPIDefinitions()
	if err != nil {
		return nil, err
	}

	results := make([]KPIResult, 0, len(defs))

	for _, def := range defs {
		current, prev := s.evaluateKPI(def, filter)
		change := 0.0
		direction := "flat"
		if prev != 0 {
			change = ((current - prev) / math.Abs(prev)) * 100
		}
		if current > prev {
			direction = "up"
		} else if current < prev {
			direction = "down"
		}

		unit := extractUnit(def.DimensionsJSON)

		results = append(results, KPIResult{
			Code:           def.Code,
			Label:          def.DisplayName,
			Value:          math.Round(current*100) / 100,
			PrevValue:      math.Round(prev*100) / 100,
			ChangePercent:  math.Round(change*100) / 100,
			TrendDirection: direction,
			Unit:           unit,
		})
	}

	return results, nil
}

// evaluateKPI computes the current and previous period value for a KPI.
// It uses well-known KPI codes with scoped queries rather than executing raw SQL
// from formula_sql (to avoid SQL injection and ensure scope enforcement).
func (s *AnalyticsService) evaluateKPI(def models.AnalyticsKPIDefinition, filter KPIFilter) (current float64, previous float64) {
	dateFrom, dateTo := resolveDateRange(filter.DateFrom, filter.DateTo)
	prevFrom, prevTo := previousPeriod(dateFrom, dateTo)

	switch {
	case strings.HasPrefix(def.Code, "master_record_count"):
		current = s.countMasterRecords(filter, dateFrom, dateTo)
		previous = s.countMasterRecords(filter, prevFrom, prevTo)

	case strings.HasPrefix(def.Code, "active_record_count"):
		current = s.countActiveMasterRecords(filter)
		previous = current // snapshot KPI, no historical comparison

	case strings.HasPrefix(def.Code, "ingestion_job_count"):
		current = s.countIngestionJobs(filter, dateFrom, dateTo)
		previous = s.countIngestionJobs(filter, prevFrom, prevTo)

	case strings.HasPrefix(def.Code, "ingestion_success_rate"):
		current = s.ingestionSuccessRate(filter, dateFrom, dateTo)
		previous = s.ingestionSuccessRate(filter, prevFrom, prevTo)

	case strings.HasPrefix(def.Code, "ingestion_failure_count"):
		current = s.countIngestionFailures(filter, dateFrom, dateTo)
		previous = s.countIngestionFailures(filter, prevFrom, prevTo)

	case strings.HasPrefix(def.Code, "report_completion_rate"):
		current = s.reportCompletionRate(filter, dateFrom, dateTo)
		previous = s.reportCompletionRate(filter, prevFrom, prevTo)

	case strings.HasPrefix(def.Code, "report_run_count"):
		current = s.countReportRuns(filter, dateFrom, dateTo)
		previous = s.countReportRuns(filter, prevFrom, prevTo)

	case strings.HasPrefix(def.Code, "version_count"):
		current = s.countVersions(filter, dateFrom, dateTo)
		previous = s.countVersions(filter, prevFrom, prevTo)

	case strings.HasPrefix(def.Code, "active_user_count"):
		current = s.countActiveUsers(filter)
		previous = current

	case strings.HasPrefix(def.Code, "audit_event_count"):
		current = s.countAuditEvents(filter, dateFrom, dateTo)
		previous = s.countAuditEvents(filter, prevFrom, prevTo)

	default:
		logging.Warn("analytics", "evaluate_kpi",
			fmt.Sprintf("unknown KPI code %q, returning zero", def.Code))
	}

	return
}

// --- individual KPI computations ---

// applyScopeToQuery adds city and department WHERE clauses to a GORM query
// when the filter carries non-wildcard scope values. The caller provides the
// column names that hold city/department in the target table.
func (s *AnalyticsService) applyScopeToQuery(q *gorm.DB, filter KPIFilter, cityCol, deptCol string) *gorm.DB {
	if filter.CityScope != "" && filter.CityScope != "*" && cityCol != "" {
		q = q.Where(cityCol+" = ?", filter.CityScope)
	}
	if filter.DeptScope != "" && filter.DeptScope != "*" && deptCol != "" {
		q = q.Where(deptCol+" = ?", filter.DeptScope)
	}
	return q
}

// scopedNodeIDs returns org-node IDs matching the user's city/department scope.
// Used to scope models that link to org_nodes rather than carrying city/dept directly.
func (s *AnalyticsService) scopedNodeIDs(filter KPIFilter) []uint {
	if (filter.CityScope == "" || filter.CityScope == "*") && (filter.DeptScope == "" || filter.DeptScope == "*") {
		return nil // no restriction
	}
	q := s.db.Model(&models.OrgNode{}).Where("is_active = ?", true)
	if filter.CityScope != "" && filter.CityScope != "*" {
		q = q.Where("city = ?", filter.CityScope)
	}
	if filter.DeptScope != "" && filter.DeptScope != "*" {
		q = q.Where("department = ?", filter.DeptScope)
	}
	var ids []uint
	q.Pluck("id", &ids)
	return ids
}

func (s *AnalyticsService) countMasterRecords(filter KPIFilter, from, to time.Time) float64 {
	var count int64
	q := s.db.Model(&models.MasterRecord{}).Where("created_at BETWEEN ? AND ?", from, to)
	// Scope via version items linked to org-scoped versions.
	if nodeIDs := s.scopedNodeIDs(filter); len(nodeIDs) > 0 {
		scopeKeys := make([]string, len(nodeIDs))
		for i, id := range nodeIDs {
			scopeKeys[i] = fmt.Sprintf("node:%d", id)
		}
		q = q.Where("id IN (SELECT master_record_id FROM master_version_items WHERE version_id IN (SELECT id FROM master_versions WHERE scope_key IN ?))", scopeKeys)
	}
	q.Count(&count)
	return float64(count)
}

func (s *AnalyticsService) countActiveMasterRecords(filter KPIFilter) float64 {
	var count int64
	q := s.db.Model(&models.MasterRecord{}).Where("status = ?", "active")
	if nodeIDs := s.scopedNodeIDs(filter); len(nodeIDs) > 0 {
		scopeKeys := make([]string, len(nodeIDs))
		for i, id := range nodeIDs {
			scopeKeys[i] = fmt.Sprintf("node:%d", id)
		}
		q = q.Where("id IN (SELECT master_record_id FROM master_version_items WHERE version_id IN (SELECT id FROM master_versions WHERE scope_key IN ?))", scopeKeys)
	}
	q.Count(&count)
	return float64(count)
}

func (s *AnalyticsService) countIngestionJobs(filter KPIFilter, from, to time.Time) float64 {
	var count int64
	q := s.db.Model(&models.IngestionJob{}).Where("created_at BETWEEN ? AND ?", from, to)
	// Scope ingestion jobs through import sources linked to scoped connectors.
	if nodeIDs := s.scopedNodeIDs(filter); len(nodeIDs) > 0 {
		scopeKeys := make([]string, len(nodeIDs))
		for i, id := range nodeIDs {
			scopeKeys[i] = fmt.Sprintf("node:%d", id)
		}
		q = q.Where("import_source_id IN (SELECT id FROM import_sources WHERE connection_json->'$.scope_key' IN ?)", scopeKeys)
	}
	q.Count(&count)
	return float64(count)
}

func (s *AnalyticsService) ingestionSuccessRate(filter KPIFilter, from, to time.Time) float64 {
	baseQ := s.db.Model(&models.IngestionJob{}).Where("created_at BETWEEN ? AND ?", from, to)
	if nodeIDs := s.scopedNodeIDs(filter); len(nodeIDs) > 0 {
		scopeKeys := make([]string, len(nodeIDs))
		for i, id := range nodeIDs {
			scopeKeys[i] = fmt.Sprintf("node:%d", id)
		}
		baseQ = baseQ.Where("import_source_id IN (SELECT id FROM import_sources WHERE connection_json->'$.scope_key' IN ?)", scopeKeys)
	}
	var total int64
	baseQ.Count(&total)
	if total == 0 {
		return 0
	}
	var completed int64
	completedQ := s.db.Model(&models.IngestionJob{}).Where("created_at BETWEEN ? AND ? AND state = ?", from, to, "completed")
	if nodeIDs := s.scopedNodeIDs(filter); len(nodeIDs) > 0 {
		scopeKeys := make([]string, len(nodeIDs))
		for i, id := range nodeIDs {
			scopeKeys[i] = fmt.Sprintf("node:%d", id)
		}
		completedQ = completedQ.Where("import_source_id IN (SELECT id FROM import_sources WHERE connection_json->'$.scope_key' IN ?)", scopeKeys)
	}
	completedQ.Count(&completed)
	return (float64(completed) / float64(total)) * 100
}

func (s *AnalyticsService) countIngestionFailures(filter KPIFilter, from, to time.Time) float64 {
	var count int64
	q := s.db.Model(&models.IngestionFailure{}).Where("created_at BETWEEN ? AND ?", from, to)
	if nodeIDs := s.scopedNodeIDs(filter); len(nodeIDs) > 0 {
		scopeKeys := make([]string, len(nodeIDs))
		for i, id := range nodeIDs {
			scopeKeys[i] = fmt.Sprintf("node:%d", id)
		}
		q = q.Where("job_id IN (SELECT id FROM ingestion_jobs WHERE import_source_id IN (SELECT id FROM import_sources WHERE connection_json->'$.scope_key' IN ?))", scopeKeys)
	}
	q.Count(&count)
	return float64(count)
}

func (s *AnalyticsService) reportCompletionRate(filter KPIFilter, from, to time.Time) float64 {
	baseQ := s.db.Model(&models.ReportRun{}).Where("created_at BETWEEN ? AND ?", from, to)
	baseQ = s.applyScopeToReportRuns(baseQ, filter)
	var total int64
	baseQ.Count(&total)
	if total == 0 {
		return 0
	}
	readyQ := s.db.Model(&models.ReportRun{}).Where("created_at BETWEEN ? AND ? AND state = ?", from, to, "ready")
	readyQ = s.applyScopeToReportRuns(readyQ, filter)
	var ready int64
	readyQ.Count(&ready)
	return (float64(ready) / float64(total)) * 100
}

func (s *AnalyticsService) countReportRuns(filter KPIFilter, from, to time.Time) float64 {
	var count int64
	q := s.db.Model(&models.ReportRun{}).Where("created_at BETWEEN ? AND ?", from, to)
	q = s.applyScopeToReportRuns(q, filter)
	q.Count(&count)
	return float64(count)
}

// applyScopeToReportRuns filters report runs by checking their schedule's scope_json
// against the user's city/department scope.
func (s *AnalyticsService) applyScopeToReportRuns(q *gorm.DB, filter KPIFilter) *gorm.DB {
	if filter.CityScope != "" && filter.CityScope != "*" {
		q = q.Where("schedule_id IN (SELECT id FROM report_schedules WHERE scope_json->'$.city' = ? OR scope_json IS NULL)", filter.CityScope)
	}
	if filter.DeptScope != "" && filter.DeptScope != "*" {
		q = q.Where("schedule_id IN (SELECT id FROM report_schedules WHERE scope_json->'$.department' = ? OR scope_json IS NULL)", filter.DeptScope)
	}
	return q
}

func (s *AnalyticsService) countVersions(filter KPIFilter, from, to time.Time) float64 {
	var count int64
	q := s.db.Model(&models.MasterVersion{}).Where("created_at BETWEEN ? AND ?", from, to)
	if nodeIDs := s.scopedNodeIDs(filter); len(nodeIDs) > 0 {
		scopeKeys := make([]string, len(nodeIDs))
		for i, id := range nodeIDs {
			scopeKeys[i] = fmt.Sprintf("node:%d", id)
		}
		q = q.Where("scope_key IN ?", scopeKeys)
	}
	q.Count(&count)
	return float64(count)
}

func (s *AnalyticsService) countActiveUsers(filter KPIFilter) float64 {
	var count int64
	q := s.db.Model(&models.User{}).Where("status = ?", "active")
	q = s.applyScopeToQuery(q, filter, "city_scope", "department_scope")
	q.Count(&count)
	return float64(count)
}

func (s *AnalyticsService) countAuditEvents(filter KPIFilter, from, to time.Time) float64 {
	var count int64
	q := s.db.Model(&models.AuditLog{}).Where("created_at BETWEEN ? AND ?", from, to)
	if filter.CityScope != "" && filter.CityScope != "*" {
		q = q.Where("scope_json->'$.city' = ?", filter.CityScope)
	}
	if filter.DeptScope != "" && filter.DeptScope != "*" {
		q = q.Where("scope_json->'$.department' = ?", filter.DeptScope)
	}
	q.Count(&count)
	return float64(count)
}

// GetTrends returns time-series data for the requested KPIs.
func (s *AnalyticsService) GetTrends(filter TrendFilter) ([]TrendSeries, error) {
	dateFrom, dateTo := resolveDateRange(filter.DateFrom, filter.DateTo)

	// Determine which KPIs to compute
	codes := filter.KPICodes
	if len(codes) == 0 {
		defs, err := s.GetKPIDefinitions()
		if err != nil {
			return nil, err
		}
		for _, d := range defs {
			codes = append(codes, d.Code)
		}
	}

	granularity := filter.Granularity
	if granularity == "" {
		granularity = "daily"
	}

	intervals := buildIntervals(dateFrom, dateTo, granularity)

	series := make([]TrendSeries, 0, len(codes))
	for _, code := range codes {
		def, err := s.GetKPIDefinitionByCode(code)
		if err != nil {
			logging.Warn("analytics", "get_trends",
				fmt.Sprintf("skipping unknown KPI code %q: %v", code, err))
			continue
		}

		points := make([]DataPoint, 0, len(intervals))
		for _, iv := range intervals {
			kf := KPIFilter{
				NodeIDs:   filter.NodeIDs,
				CityScope: filter.CityScope,
				DeptScope: filter.DeptScope,
				DateFrom:  iv.From.Format("2006-01-02"),
				DateTo:    iv.To.Format("2006-01-02"),
			}
			val, _ := s.evaluateKPI(*def, kf)
			points = append(points, DataPoint{
				Date:  iv.From.Format("2006-01-02"),
				Value: math.Round(val*100) / 100,
			})
		}

		series = append(series, TrendSeries{
			Code:   def.Code,
			Label:  def.DisplayName,
			Points: points,
		})
	}

	return series, nil
}

// --- helpers ---

type interval struct {
	From time.Time
	To   time.Time
}

func buildIntervals(from, to time.Time, granularity string) []interval {
	var intervals []interval
	cursor := from
	for cursor.Before(to) {
		var end time.Time
		switch granularity {
		case "weekly":
			end = cursor.AddDate(0, 0, 7)
		case "monthly":
			end = cursor.AddDate(0, 1, 0)
		default: // daily
			end = cursor.AddDate(0, 0, 1)
		}
		if end.After(to) {
			end = to
		}
		intervals = append(intervals, interval{From: cursor, To: end})
		cursor = end
	}
	return intervals
}

func resolveDateRange(from, to string) (time.Time, time.Time) {
	layout := "2006-01-02"
	now := time.Now()

	dateFrom, err := time.Parse(layout, from)
	if err != nil {
		dateFrom = now.AddDate(0, -1, 0) // default: last 30 days
	}

	dateTo, err := time.Parse(layout, to)
	if err != nil {
		dateTo = now
	}

	return dateFrom, dateTo
}

func previousPeriod(from, to time.Time) (time.Time, time.Time) {
	duration := to.Sub(from)
	return from.Add(-duration), from
}

func extractUnit(dimensionsJSON string) string {
	// Simple extraction; defaults based on common patterns
	lower := strings.ToLower(dimensionsJSON)
	if strings.Contains(lower, "percent") || strings.Contains(lower, "rate") {
		return "%"
	}
	if strings.Contains(lower, "count") {
		return "count"
	}
	return "count"
}
