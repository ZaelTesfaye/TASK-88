package ingestion

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ValidationRule defines a field validation.
type ValidationRule struct {
	Field   string `json:"field"`
	Type    string `json:"type"` // regex, required, range, enum
	Pattern string `json:"pattern,omitempty"`
	Min     string `json:"min,omitempty"`
	Max     string `json:"max,omitempty"`
	Values  string `json:"values,omitempty"` // comma-separated enum values
	Message string `json:"message"`
}

// ValidationError holds details about a single validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s (value=%q)", e.Field, e.Message, e.Value)
}

// DefaultValidationRules returns built-in rules per entity type.
func DefaultValidationRules(entityType string) []ValidationRule {
	switch strings.ToLower(entityType) {
	case "sku":
		return []ValidationRule{
			{Field: "code", Type: "required", Message: "SKU code is required"},
			{Field: "code", Type: "regex", Pattern: `^[A-Z0-9]{6,20}$`, Message: "SKU code must be 6-20 uppercase alphanumeric characters"},
			{Field: "name", Type: "required", Message: "SKU name is required"},
		}
	case "season":
		return []ValidationRule{
			{Field: "code", Type: "required", Message: "season code is required"},
			{Field: "code", Type: "regex", Pattern: `^(SS|FW)[0-9]{4}$`, Message: "season code must match format SS/FW followed by 4 digits (e.g. SS2025, FW2024)"},
			{Field: "name", Type: "required", Message: "season name is required"},
		}
	case "supplier":
		return []ValidationRule{
			{Field: "name", Type: "required", Message: "supplier name is required"},
			{Field: "phone", Type: "regex", Pattern: `^\([0-9]{3}\) [0-9]{3}-[0-9]{4}$`, Message: "phone must match format (XXX) XXX-XXXX"},
			{Field: "email", Type: "regex", Pattern: `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`, Message: "email must be a valid email address"},
		}
	case "customer":
		return []ValidationRule{
			{Field: "name", Type: "required", Message: "customer name is required"},
			{Field: "code", Type: "required", Message: "customer code is required"},
			{Field: "code", Type: "regex", Pattern: `^[A-Z0-9]{4,20}$`, Message: "customer code must be 4-20 uppercase alphanumeric characters"},
		}
	default:
		return []ValidationRule{}
	}
}

// ValidateRecord validates a single record against rules.
func ValidateRecord(record map[string]string, rules []ValidationRule) []ValidationError {
	var errs []ValidationError

	for _, rule := range rules {
		value := strings.TrimSpace(record[rule.Field])

		switch rule.Type {
		case "required":
			if value == "" {
				errs = append(errs, ValidationError{
					Field:   rule.Field,
					Value:   value,
					Rule:    "required",
					Message: rule.Message,
				})
			}

		case "regex":
			if value == "" {
				continue // Skip regex validation for empty values; use "required" for that.
			}
			re, err := regexp.Compile(rule.Pattern)
			if err != nil {
				errs = append(errs, ValidationError{
					Field:   rule.Field,
					Value:   value,
					Rule:    "regex",
					Message: fmt.Sprintf("invalid regex pattern in rule: %s", rule.Pattern),
				})
				continue
			}
			if !re.MatchString(value) {
				errs = append(errs, ValidationError{
					Field:   rule.Field,
					Value:   value,
					Rule:    "regex",
					Message: rule.Message,
				})
			}

		case "range":
			if value == "" {
				continue
			}
			numVal, err := strconv.ParseFloat(value, 64)
			if err != nil {
				errs = append(errs, ValidationError{
					Field:   rule.Field,
					Value:   value,
					Rule:    "range",
					Message: fmt.Sprintf("value must be a number for range validation"),
				})
				continue
			}
			if rule.Min != "" {
				minVal, err := strconv.ParseFloat(rule.Min, 64)
				if err == nil && numVal < minVal {
					errs = append(errs, ValidationError{
						Field:   rule.Field,
						Value:   value,
						Rule:    "range",
						Message: rule.Message,
					})
				}
			}
			if rule.Max != "" {
				maxVal, err := strconv.ParseFloat(rule.Max, 64)
				if err == nil && numVal > maxVal {
					errs = append(errs, ValidationError{
						Field:   rule.Field,
						Value:   value,
						Rule:    "range",
						Message: rule.Message,
					})
				}
			}

		case "enum":
			if value == "" {
				continue
			}
			allowed := strings.Split(rule.Values, ",")
			found := false
			for _, v := range allowed {
				if strings.TrimSpace(v) == value {
					found = true
					break
				}
			}
			if !found {
				errs = append(errs, ValidationError{
					Field:   rule.Field,
					Value:   value,
					Rule:    "enum",
					Message: rule.Message,
				})
			}
		}
	}

	return errs
}

// ValidateImportFile validates file type, encoding, size, and template columns.
func ValidateImportFile(fileName string, data []byte, entityType string) error {
	// Check file size: max 50MB.
	const maxFileSize = 50 * 1024 * 1024
	if len(data) > maxFileSize {
		return fmt.Errorf("file size exceeds maximum allowed size of 50MB")
	}

	// Check file extension.
	ext := strings.ToLower(fileName)
	isCSV := strings.HasSuffix(ext, ".csv")
	isXLSX := strings.HasSuffix(ext, ".xlsx")
	if !isCSV && !isXLSX {
		return fmt.Errorf("unsupported file type: only CSV and XLSX files are accepted")
	}

	// Check for UTF-8 encoding by looking for invalid byte sequences.
	if isCSV {
		// Strip UTF-8 BOM if present.
		cleaned := stripBOM(data)
		if !isValidUTF8(cleaned) {
			return fmt.Errorf("file encoding is not valid UTF-8")
		}
	}

	// Validate expected columns.
	expectedCols := ExpectedColumns(entityType)
	if len(expectedCols) == 0 {
		return nil // No column validation for unknown entity types.
	}

	var actualCols []string
	var err error
	if isCSV {
		_, actualCols, err = ParseCSV(data)
	} else {
		_, actualCols, err = ParseXLSX(data)
	}
	if err != nil {
		return fmt.Errorf("failed to parse file headers: %w", err)
	}

	// Normalize actual column names.
	actualSet := make(map[string]bool, len(actualCols))
	for _, col := range actualCols {
		actualSet[strings.ToLower(strings.TrimSpace(col))] = true
	}

	var missing []string
	for _, expected := range expectedCols {
		if !actualSet[strings.ToLower(expected)] {
			missing = append(missing, expected)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required columns: %s", strings.Join(missing, ", "))
	}

	return nil
}

// ParseCSV parses CSV data into records, returning records and header column names.
func ParseCSV(data []byte) ([]map[string]string, []string, error) {
	cleaned := stripBOM(data)

	reader := csv.NewReader(bytes.NewReader(cleaned))
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	// Trim headers.
	for i := range headers {
		headers[i] = strings.TrimSpace(headers[i])
	}

	var records []map[string]string
	lineNum := 1
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read CSV row %d: %w", lineNum+1, err)
		}
		lineNum++

		record := make(map[string]string, len(headers))
		for i, header := range headers {
			if i < len(row) {
				record[header] = row[i]
			} else {
				record[header] = ""
			}
		}
		records = append(records, record)
	}

	return records, headers, nil
}

// ParseXLSX parses XLSX data into records, returning records and header column names.
func ParseXLSX(data []byte) ([]map[string]string, []string, error) {
	reader := bytes.NewReader(data)
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open XLSX data: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, nil, fmt.Errorf("no sheets found in XLSX file")
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read XLSX rows: %w", err)
	}

	if len(rows) < 1 {
		return nil, nil, fmt.Errorf("XLSX file has no rows")
	}

	headers := rows[0]
	for i := range headers {
		headers[i] = strings.TrimSpace(headers[i])
	}

	var records []map[string]string
	for _, row := range rows[1:] {
		record := make(map[string]string, len(headers))
		for i, header := range headers {
			if i < len(row) {
				record[header] = row[i]
			} else {
				record[header] = ""
			}
		}
		records = append(records, record)
	}

	return records, headers, nil
}

// ExpectedColumns returns expected column names per entity type.
func ExpectedColumns(entityType string) []string {
	switch strings.ToLower(entityType) {
	case "sku":
		return []string{"code", "name", "category", "status"}
	case "season":
		return []string{"code", "name", "start_date", "end_date"}
	case "supplier":
		return []string{"name", "code", "phone", "email", "address"}
	case "customer":
		return []string{"name", "code", "email"}
	default:
		return nil
	}
}

// stripBOM removes a UTF-8 BOM (byte order mark) from the data if present.
func stripBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}

// isValidUTF8 checks if the data is valid UTF-8.
func isValidUTF8(data []byte) bool {
	i := 0
	for i < len(data) {
		if data[i] < 0x80 {
			i++
			continue
		}

		var size int
		switch {
		case data[i]&0xE0 == 0xC0:
			size = 2
		case data[i]&0xF0 == 0xE0:
			size = 3
		case data[i]&0xF8 == 0xF0:
			size = 4
		default:
			return false
		}

		if i+size > len(data) {
			return false
		}
		for j := 1; j < size; j++ {
			if data[i+j]&0xC0 != 0x80 {
				return false
			}
		}
		i += size
	}
	return true
}
