package masterdata

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	appErrors "backend/internal/errors"
	"backend/internal/models"
)

// Entity type constants.
const (
	EntitySKU      = "sku"
	EntityColor    = "color"
	EntitySize     = "size"
	EntitySeason   = "season"
	EntityBrand    = "brand"
	EntitySupplier = "supplier"
	EntityCustomer = "customer"
)

// validEntityTypes lists all allowed entity types.
var validEntityTypes = map[string]bool{
	EntitySKU:      true,
	EntityColor:    true,
	EntitySize:     true,
	EntitySeason:   true,
	EntityBrand:    true,
	EntitySupplier: true,
	EntityCustomer: true,
}

// Validation patterns per entity type.
var (
	skuCodePattern      = regexp.MustCompile(`^[A-Z0-9]{6,20}$`)
	seasonCodePattern   = regexp.MustCompile(`^(SS|FW)[0-9]{4}$`)
	supplierPhonePattern = regexp.MustCompile(`^\([0-9]{3}\) [0-9]{3}-[0-9]{4}$`)
)

// Expected template columns per entity type for import.
var entityColumns = map[string][]string{
	EntitySKU:      {"code", "name", "description", "category"},
	EntityColor:    {"code", "name", "hex_value"},
	EntitySize:     {"code", "name", "sort_order"},
	EntitySeason:   {"code", "name", "start_date", "end_date"},
	EntityBrand:    {"code", "name", "description"},
	EntitySupplier: {"code", "name", "phone", "email", "address"},
	EntityCustomer: {"code", "name", "phone", "email", "address"},
}

const maxImportFileSize = 50 * 1024 * 1024 // 50MB

// ListFilter holds parameters for listing master records.
type ListFilter struct {
	Search    string
	Status    string // active, inactive, all
	SortBy    string
	SortOrder string // asc, desc
	Page      int
	PageSize  int
	CityScope string
	DeptScope string
	NodeIDs   []uint // for context scope filtering
}

// PaginatedResult holds a paginated list of master records.
type PaginatedResult struct {
	Items      []models.MasterRecord `json:"items"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalPages int                   `json:"total_pages"`
}

// CreateRecordRequest holds the data to create a master record.
type CreateRecordRequest struct {
	NaturalKey  string `json:"natural_key" binding:"required"`
	PayloadJSON string `json:"payload_json"`
}

// UpdateRecordRequest holds the data to update a master record.
type UpdateRecordRequest struct {
	NaturalKey  *string `json:"natural_key"`
	PayloadJSON *string `json:"payload_json"`
	Status      *string `json:"status"`
}

// ImportResult holds the result of a bulk import operation.
type ImportResult struct {
	TotalRows    int           `json:"total_rows"`
	SuccessCount int           `json:"success_count"`
	ErrorCount   int           `json:"error_count"`
	Errors       []ImportError `json:"errors"`
}

// ImportError describes a single import row error.
type ImportError struct {
	Row     int    `json:"row"`
	Column  string `json:"column"`
	Message string `json:"message"`
	Value   string `json:"value"`
}

// MasterService provides master data management operations.
type MasterService struct {
	db *gorm.DB
}

// NewMasterService creates a new MasterService.
func NewMasterService(db *gorm.DB) *MasterService {
	return &MasterService{db: db}
}

// ListRecords returns paginated, filtered, sorted records for an entity type.
func (s *MasterService) ListRecords(entityType string, filter ListFilter) (*PaginatedResult, error) {
	if !validEntityTypes[entityType] {
		return nil, appErrors.ValidationError(
			fmt.Sprintf("invalid entity type %q", entityType),
			map[string]interface{}{"valid_types": getValidEntityTypes()},
		)
	}

	// Defaults.
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 25
	}
	if filter.PageSize > 200 {
		filter.PageSize = 200
	}
	if filter.SortOrder == "" {
		filter.SortOrder = "asc"
	}
	if filter.SortOrder != "asc" && filter.SortOrder != "desc" {
		filter.SortOrder = "asc"
	}

	query := s.db.Model(&models.MasterRecord{}).Where("entity_type = ?", entityType)

	// Status filter.
	switch filter.Status {
	case "active":
		query = query.Where("status = ?", "active")
	case "inactive":
		query = query.Where("status = ?", "inactive")
	case "all", "":
		// No status filter.
	default:
		query = query.Where("status = ?", filter.Status)
	}

	// Search filter.
	if filter.Search != "" {
		searchTerm := "%" + filter.Search + "%"
		query = query.Where("(natural_key LIKE ? OR payload_json LIKE ?)", searchTerm, searchTerm)
	}

	// Context scope filtering by node IDs.
	if len(filter.NodeIDs) > 0 {
		query = query.Where("id IN (SELECT master_record_id FROM master_version_items WHERE version_id IN (SELECT id FROM master_versions WHERE scope_key IN ? AND state = 'active'))",
			scopeKeysFromNodeIDs(filter.NodeIDs))
	} else if filter.CityScope != "" && filter.CityScope != "*" || filter.DeptScope != "" && filter.DeptScope != "*" {
		// Fallback: when no org-context NodeIDs are available, scope via org_nodes
		// that match the user's city/department and link through versions.
		subQ := s.db.Table("org_nodes").Select("id").Where("is_active = ?", true)
		if filter.CityScope != "" && filter.CityScope != "*" {
			subQ = subQ.Where("city = ?", filter.CityScope)
		}
		if filter.DeptScope != "" && filter.DeptScope != "*" {
			subQ = subQ.Where("department = ?", filter.DeptScope)
		}
		var nodeIDs []uint
		subQ.Pluck("id", &nodeIDs)
		if len(nodeIDs) > 0 {
			query = query.Where("id IN (SELECT master_record_id FROM master_version_items WHERE version_id IN (SELECT id FROM master_versions WHERE scope_key IN ? AND state = 'active'))",
				scopeKeysFromNodeIDs(nodeIDs))
		}
	}

	// Count total matching.
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, appErrors.InternalError(fmt.Sprintf("failed to count records: %v", err))
	}

	// Sorting.
	sortColumn := "natural_key"
	allowedSortColumns := map[string]string{
		"natural_key": "natural_key",
		"status":      "status",
		"created_at":  "created_at",
		"updated_at":  "updated_at",
		"id":          "id",
	}
	if col, ok := allowedSortColumns[filter.SortBy]; ok {
		sortColumn = col
	}
	query = query.Order(fmt.Sprintf("%s %s", sortColumn, filter.SortOrder))

	// Pagination.
	offset := (filter.Page - 1) * filter.PageSize
	var items []models.MasterRecord
	if err := query.Offset(offset).Limit(filter.PageSize).Find(&items).Error; err != nil {
		return nil, appErrors.InternalError(fmt.Sprintf("failed to list records: %v", err))
	}
	if items == nil {
		items = []models.MasterRecord{}
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.PageSize)))

	return &PaginatedResult{
		Items:      items,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// CreateRecord creates a new master record with duplicate checking.
func (s *MasterService) CreateRecord(entityType string, req CreateRecordRequest) (*models.MasterRecord, error) {
	if !validEntityTypes[entityType] {
		return nil, appErrors.ValidationError(
			fmt.Sprintf("invalid entity type %q", entityType),
			map[string]interface{}{"valid_types": getValidEntityTypes()},
		)
	}

	naturalKey := strings.TrimSpace(req.NaturalKey)
	if naturalKey == "" {
		return nil, appErrors.ValidationError("natural_key is required", nil)
	}

	// Entity-specific validation on natural_key.
	if err := validateNaturalKey(entityType, naturalKey); err != nil {
		return nil, err
	}

	// Check exact duplicate.
	var existingCount int64
	if err := s.db.Model(&models.MasterRecord{}).
		Where("entity_type = ? AND natural_key = ?", entityType, naturalKey).
		Count(&existingCount).Error; err != nil {
		return nil, appErrors.InternalError(fmt.Sprintf("failed to check duplicates: %v", err))
	}
	if existingCount > 0 {
		return nil, appErrors.Conflict(
			fmt.Sprintf("a %s record with key %q already exists", entityType, naturalKey),
			map[string]interface{}{"natural_key": naturalKey},
		)
	}

	record := models.MasterRecord{
		EntityType:  entityType,
		NaturalKey:  naturalKey,
		PayloadJSON: req.PayloadJSON,
		Status:      "active",
	}

	if err := s.db.Create(&record).Error; err != nil {
		return nil, appErrors.InternalError(fmt.Sprintf("failed to create record: %v", err))
	}

	return &record, nil
}

// UpdateRecord updates an existing master record.
func (s *MasterService) UpdateRecord(id uint, req UpdateRecordRequest) (*models.MasterRecord, error) {
	var record models.MasterRecord
	if err := s.db.First(&record, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, appErrors.NotFound(fmt.Sprintf("master record %d not found", id))
		}
		return nil, appErrors.InternalError(fmt.Sprintf("failed to load record: %v", err))
	}

	if req.NaturalKey != nil {
		newKey := strings.TrimSpace(*req.NaturalKey)
		if newKey == "" {
			return nil, appErrors.ValidationError("natural_key cannot be empty", nil)
		}

		// Entity-specific validation.
		if err := validateNaturalKey(record.EntityType, newKey); err != nil {
			return nil, err
		}

		// Check uniqueness (exclude self).
		var count int64
		if err := s.db.Model(&models.MasterRecord{}).
			Where("entity_type = ? AND natural_key = ? AND id != ?", record.EntityType, newKey, id).
			Count(&count).Error; err != nil {
			return nil, appErrors.InternalError(fmt.Sprintf("failed to check uniqueness: %v", err))
		}
		if count > 0 {
			return nil, appErrors.Conflict(
				fmt.Sprintf("a %s record with key %q already exists", record.EntityType, newKey),
				map[string]interface{}{"natural_key": newKey},
			)
		}
		record.NaturalKey = newKey
	}

	if req.PayloadJSON != nil {
		record.PayloadJSON = *req.PayloadJSON
	}

	if req.Status != nil {
		status := strings.TrimSpace(*req.Status)
		if status != "active" && status != "inactive" {
			return nil, appErrors.ValidationError("status must be 'active' or 'inactive'", nil)
		}
		record.Status = status
	}

	if err := s.db.Save(&record).Error; err != nil {
		return nil, appErrors.InternalError(fmt.Sprintf("failed to update record: %v", err))
	}

	return &record, nil
}

// DeactivateRecord soft-deactivates a record with a reason.
func (s *MasterService) DeactivateRecord(id uint, reason string, actorID uint) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return appErrors.ValidationError("deactivation reason is required", nil)
	}

	var record models.MasterRecord
	if err := s.db.First(&record, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return appErrors.NotFound(fmt.Sprintf("master record %d not found", id))
		}
		return appErrors.InternalError(fmt.Sprintf("failed to load record: %v", err))
	}

	if record.Status == "inactive" {
		return appErrors.ValidationError("record is already inactive", nil)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Set status to inactive.
		if err := tx.Model(&record).Update("status", "inactive").Error; err != nil {
			return appErrors.InternalError(fmt.Sprintf("failed to deactivate record: %v", err))
		}

		// Create deactivation event.
		event := models.DeactivationEvent{
			RecordID:    id,
			Reason:      reason,
			ActorUserID: actorID,
		}
		if err := tx.Create(&event).Error; err != nil {
			return appErrors.InternalError(fmt.Sprintf("failed to create deactivation event: %v", err))
		}

		return nil
	})
}

// CheckDuplicates checks for duplicate records using normalized matching.
func (s *MasterService) CheckDuplicates(entityType string, naturalKey string) ([]models.MasterRecord, error) {
	if !validEntityTypes[entityType] {
		return nil, appErrors.ValidationError(
			fmt.Sprintf("invalid entity type %q", entityType),
			map[string]interface{}{"valid_types": getValidEntityTypes()},
		)
	}

	normalized := normalizeKey(naturalKey)
	if normalized == "" {
		return nil, appErrors.ValidationError("key is required for duplicate check", nil)
	}

	// Search for near-matches: exact match after normalization, or LIKE match.
	var records []models.MasterRecord
	searchTerm := "%" + normalized + "%"

	if err := s.db.Where("entity_type = ? AND (UPPER(REPLACE(REPLACE(natural_key, ' ', ''), '-', '')) LIKE ? OR UPPER(natural_key) = ?)",
		entityType, searchTerm, normalized).
		Limit(20).
		Find(&records).Error; err != nil {
		return nil, appErrors.InternalError(fmt.Sprintf("failed to check duplicates: %v", err))
	}

	if records == nil {
		records = []models.MasterRecord{}
	}

	return records, nil
}

// ImportRecords handles bulk CSV/XLSX import.
func (s *MasterService) ImportRecords(entityType string, fileData []byte, fileName string) (*ImportResult, error) {
	if !validEntityTypes[entityType] {
		return nil, appErrors.ValidationError(
			fmt.Sprintf("invalid entity type %q", entityType),
			map[string]interface{}{"valid_types": getValidEntityTypes()},
		)
	}

	// Check file size.
	if len(fileData) > maxImportFileSize {
		return nil, appErrors.ValidationError(
			fmt.Sprintf("file size exceeds maximum of %dMB", maxImportFileSize/(1024*1024)),
			nil,
		)
	}

	// Check UTF-8 encoding.
	if !utf8.Valid(fileData) {
		return nil, appErrors.ValidationError("file is not valid UTF-8 encoded", nil)
	}

	// Detect file type by extension.
	ext := strings.ToLower(filepath.Ext(fileName))
	var rows [][]string
	var parseErr error

	switch ext {
	case ".csv":
		rows, parseErr = parseCSV(fileData)
	case ".xlsx":
		rows, parseErr = parseXLSX(fileData)
	default:
		return nil, appErrors.ValidationError(
			fmt.Sprintf("unsupported file type %q; use .csv or .xlsx", ext),
			nil,
		)
	}
	if parseErr != nil {
		return nil, appErrors.ValidationError(fmt.Sprintf("failed to parse file: %v", parseErr), nil)
	}

	if len(rows) < 2 {
		return nil, appErrors.ValidationError("file must contain a header row and at least one data row", nil)
	}

	// Validate template columns.
	expectedCols := entityColumns[entityType]
	headerRow := rows[0]
	normalizedHeaders := make([]string, len(headerRow))
	for i, h := range headerRow {
		normalizedHeaders[i] = strings.TrimSpace(strings.ToLower(h))
	}

	if err := validateColumns(normalizedHeaders, expectedCols); err != nil {
		return nil, err
	}

	// Build column index map.
	colIndex := make(map[string]int)
	for i, h := range normalizedHeaders {
		colIndex[h] = i
	}

	result := &ImportResult{
		TotalRows: len(rows) - 1,
		Errors:    []ImportError{},
	}

	// Process each data row.
	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		rowNum := rowIdx + 1 // 1-based, accounting for header.

		// Skip completely empty rows.
		allEmpty := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			result.TotalRows--
			continue
		}

		rowErrors := validateImportRow(entityType, row, colIndex, rowNum)
		if len(rowErrors) > 0 {
			result.Errors = append(result.Errors, rowErrors...)
			result.ErrorCount++
			continue
		}

		// Extract natural_key from the "code" column.
		codeIdx, hasCode := colIndex["code"]
		if !hasCode || codeIdx >= len(row) {
			result.Errors = append(result.Errors, ImportError{
				Row:     rowNum,
				Column:  "code",
				Message: "missing code column",
			})
			result.ErrorCount++
			continue
		}

		naturalKey := strings.TrimSpace(row[codeIdx])

		// Build payload JSON from all columns.
		payloadParts := make([]string, 0)
		for _, col := range expectedCols {
			idx, ok := colIndex[col]
			if ok && idx < len(row) {
				val := strings.TrimSpace(row[idx])
				// Escape JSON special characters.
				val = strings.ReplaceAll(val, `\`, `\\`)
				val = strings.ReplaceAll(val, `"`, `\"`)
				payloadParts = append(payloadParts, fmt.Sprintf(`"%s":"%s"`, col, val))
			}
		}
		payloadJSON := "{" + strings.Join(payloadParts, ",") + "}"

		// Check for existing record with same natural key.
		var existingCount int64
		if err := s.db.Model(&models.MasterRecord{}).
			Where("entity_type = ? AND natural_key = ?", entityType, naturalKey).
			Count(&existingCount).Error; err != nil {
			result.Errors = append(result.Errors, ImportError{
				Row:     rowNum,
				Column:  "code",
				Message: fmt.Sprintf("database error checking duplicates: %v", err),
				Value:   naturalKey,
			})
			result.ErrorCount++
			continue
		}
		if existingCount > 0 {
			result.Errors = append(result.Errors, ImportError{
				Row:     rowNum,
				Column:  "code",
				Message: fmt.Sprintf("record with key %q already exists", naturalKey),
				Value:   naturalKey,
			})
			result.ErrorCount++
			continue
		}

		// Create the record.
		record := models.MasterRecord{
			EntityType:  entityType,
			NaturalKey:  naturalKey,
			PayloadJSON: payloadJSON,
			Status:      "active",
		}
		if err := s.db.Create(&record).Error; err != nil {
			result.Errors = append(result.Errors, ImportError{
				Row:     rowNum,
				Column:  "",
				Message: fmt.Sprintf("failed to create record: %v", err),
			})
			result.ErrorCount++
			continue
		}

		result.SuccessCount++
	}

	return result, nil
}

// validateNaturalKey validates the natural key based on entity-specific rules.
func validateNaturalKey(entityType, key string) *appErrors.AppError {
	switch entityType {
	case EntitySKU:
		if !skuCodePattern.MatchString(key) {
			return appErrors.ValidationError(
				"SKU code must match pattern ^[A-Z0-9]{6,20}$ (6-20 uppercase alphanumeric characters)",
				map[string]interface{}{"pattern": "^[A-Z0-9]{6,20}$", "value": key},
			)
		}
	case EntitySeason:
		if !seasonCodePattern.MatchString(key) {
			return appErrors.ValidationError(
				"season code must match pattern ^(SS|FW)[0-9]{4}$ (e.g. SS2025, FW2024)",
				map[string]interface{}{"pattern": "^(SS|FW)[0-9]{4}$", "value": key},
			)
		}
	}
	return nil
}

// validateImportRow validates a single import row against entity-specific rules.
func validateImportRow(entityType string, row []string, colIndex map[string]int, rowNum int) []ImportError {
	var errs []ImportError

	// Ensure code column exists and is not empty.
	codeIdx, hasCode := colIndex["code"]
	if !hasCode || codeIdx >= len(row) || strings.TrimSpace(row[codeIdx]) == "" {
		errs = append(errs, ImportError{
			Row:     rowNum,
			Column:  "code",
			Message: "code is required",
		})
		return errs
	}
	code := strings.TrimSpace(row[codeIdx])

	// Ensure name column exists and is not empty.
	nameIdx, hasName := colIndex["name"]
	if !hasName || nameIdx >= len(row) || strings.TrimSpace(row[nameIdx]) == "" {
		errs = append(errs, ImportError{
			Row:     rowNum,
			Column:  "name",
			Message: "name is required",
		})
	}

	// Entity-specific validation.
	switch entityType {
	case EntitySKU:
		if !skuCodePattern.MatchString(code) {
			errs = append(errs, ImportError{
				Row:     rowNum,
				Column:  "code",
				Message: "SKU code must match ^[A-Z0-9]{6,20}$",
				Value:   code,
			})
		}
	case EntitySeason:
		if !seasonCodePattern.MatchString(code) {
			errs = append(errs, ImportError{
				Row:     rowNum,
				Column:  "code",
				Message: "season code must match ^(SS|FW)[0-9]{4}$",
				Value:   code,
			})
		}
	case EntitySupplier:
		phoneIdx, hasPhone := colIndex["phone"]
		if hasPhone && phoneIdx < len(row) {
			phone := strings.TrimSpace(row[phoneIdx])
			if phone != "" && !supplierPhonePattern.MatchString(phone) {
				errs = append(errs, ImportError{
					Row:     rowNum,
					Column:  "phone",
					Message: `supplier phone must match ^\([0-9]{3}\) [0-9]{3}-[0-9]{4}$`,
					Value:   phone,
				})
			}
		}
	case EntityCustomer:
		phoneIdx, hasPhone := colIndex["phone"]
		if hasPhone && phoneIdx < len(row) {
			phone := strings.TrimSpace(row[phoneIdx])
			if phone != "" && !supplierPhonePattern.MatchString(phone) {
				errs = append(errs, ImportError{
					Row:     rowNum,
					Column:  "phone",
					Message: `customer phone must match ^\([0-9]{3}\) [0-9]{3}-[0-9]{4}$`,
					Value:   phone,
				})
			}
		}
	}

	return errs
}

// parseCSV parses CSV file data into rows.
func parseCSV(data []byte) ([][]string, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	var rows [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("CSV parse error: %w", err)
		}
		rows = append(rows, record)
	}
	return rows, nil
}

// parseXLSX parses XLSX file data into rows.
func parseXLSX(data []byte) ([][]string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("XLSX parse error: %w", err)
	}
	defer f.Close()

	// Use the first sheet.
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("XLSX file has no sheets")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read XLSX rows: %w", err)
	}

	return rows, nil
}

// validateColumns checks that the file header contains all expected columns.
func validateColumns(headers, expected []string) *appErrors.AppError {
	headerSet := make(map[string]bool, len(headers))
	for _, h := range headers {
		headerSet[h] = true
	}

	var missing []string
	for _, col := range expected {
		if !headerSet[col] {
			missing = append(missing, col)
		}
	}

	if len(missing) > 0 {
		return appErrors.ValidationError(
			fmt.Sprintf("missing required columns: %s", strings.Join(missing, ", ")),
			map[string]interface{}{
				"missing_columns":  missing,
				"expected_columns": expected,
				"found_columns":    headers,
			},
		)
	}
	return nil
}

// normalizeKey normalizes a key for duplicate comparison.
// Trims whitespace, converts to uppercase, and removes extra punctuation and spaces.
func normalizeKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.ToUpper(key)
	key = strings.ReplaceAll(key, " ", "")
	key = strings.ReplaceAll(key, "-", "")
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, ".", "")
	return key
}

// scopeKeysFromNodeIDs converts node IDs to scope key strings.
func scopeKeysFromNodeIDs(nodeIDs []uint) []string {
	keys := make([]string, len(nodeIDs))
	for i, id := range nodeIDs {
		keys[i] = fmt.Sprintf("node:%d", id)
	}
	return keys
}

// getValidEntityTypes returns a sorted list of valid entity types.
func getValidEntityTypes() []string {
	return []string{EntityBrand, EntityColor, EntityCustomer, EntitySeason, EntitySize, EntitySKU, EntitySupplier}
}
