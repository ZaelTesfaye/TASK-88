package ingestion

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/xuri/excelize/v2"

	"backend/internal/models"
)

// ConnectorType constants.
const (
	ConnectorFolder = "folder"
	ConnectorShare  = "network_share"
	ConnectorDB     = "database"
)

// Connector interface that all connectors must implement.
type Connector interface {
	// Type returns the connector type.
	Type() string
	// Capabilities returns what this connector can do.
	Capabilities() []string
	// ValidateConfig validates the connector configuration.
	ValidateConfig(config map[string]interface{}) error
	// HealthCheck tests connectivity.
	HealthCheck() (*HealthResult, error)
	// Pull fetches records starting from cursor.
	Pull(cursor string, batchSize int) (*PullResult, error)
	// AcknowledgeCheckpoint confirms a checkpoint was saved.
	AcknowledgeCheckpoint(cursor string) error
}

// HealthResult holds the result of a health check.
type HealthResult struct {
	Healthy   bool      `json:"healthy"`
	Message   string    `json:"message"`
	CheckedAt time.Time `json:"checked_at"`
}

// PullResult holds the result of a pull operation.
type PullResult struct {
	Records    []map[string]interface{} `json:"records"`
	NextCursor string                   `json:"next_cursor"`
	HasMore    bool                     `json:"has_more"`
}

// ConnectorFactory creates connectors from definitions.
type ConnectorFactory struct{}

// NewConnectorFactory creates a new ConnectorFactory.
func NewConnectorFactory() *ConnectorFactory {
	return &ConnectorFactory{}
}

// Create instantiates a connector based on the definition's connector type.
func (f *ConnectorFactory) Create(def models.ConnectorDefinition, config map[string]interface{}) (Connector, error) {
	switch def.ConnectorType {
	case ConnectorFolder:
		return NewFolderConnector(config)
	case ConnectorShare:
		return NewShareConnector(config)
	case ConnectorDB:
		return NewDatabaseConnector(config)
	default:
		return nil, fmt.Errorf("unsupported connector type: %s", def.ConnectorType)
	}
}

// CreateFromSource instantiates a connector using an ImportSource's configuration.
func (f *ConnectorFactory) CreateFromSource(source models.ImportSource, connConfig map[string]interface{}) (Connector, error) {
	switch source.SourceType {
	case ConnectorFolder:
		return NewFolderConnector(connConfig)
	case ConnectorShare:
		return NewShareConnector(connConfig)
	case ConnectorDB:
		return NewDatabaseConnector(connConfig)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", source.SourceType)
	}
}

// ---------------------------------------------------------------------------
// FolderConnector: reads CSV/XLSX files from a local directory path
// ---------------------------------------------------------------------------

// FolderConnector reads data from local filesystem directories.
type FolderConnector struct {
	path          string
	filePattern   string
	records       []map[string]interface{}
	loaded        bool
	lastCheckpoint string
}

// NewFolderConnector creates a FolderConnector from config.
// Required config keys: "path" (string). Optional: "file_pattern" (string, default "*.csv").
func NewFolderConnector(config map[string]interface{}) (*FolderConnector, error) {
	path, ok := config["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("folder connector requires 'path' in configuration")
	}

	pattern := "*.csv"
	if p, ok := config["file_pattern"].(string); ok && p != "" {
		pattern = p
	}

	return &FolderConnector{
		path:        path,
		filePattern: pattern,
	}, nil
}

func (c *FolderConnector) Type() string {
	return ConnectorFolder
}

func (c *FolderConnector) Capabilities() []string {
	return []string{"pull", "incremental", "backfill", "csv", "xlsx"}
}

func (c *FolderConnector) ValidateConfig(config map[string]interface{}) error {
	path, ok := config["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("'path' is required and must be a non-empty string")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path does not exist or is inaccessible: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path must be a directory")
	}
	return nil
}

func (c *FolderConnector) HealthCheck() (*HealthResult, error) {
	now := time.Now()
	info, err := os.Stat(c.path)
	if err != nil {
		return &HealthResult{
			Healthy:   false,
			Message:   fmt.Sprintf("cannot access path: %v", err),
			CheckedAt: now,
		}, nil
	}
	if !info.IsDir() {
		return &HealthResult{
			Healthy:   false,
			Message:   "path is not a directory",
			CheckedAt: now,
		}, nil
	}
	return &HealthResult{
		Healthy:   true,
		Message:   fmt.Sprintf("directory accessible: %s", c.path),
		CheckedAt: now,
	}, nil
}

func (c *FolderConnector) Pull(cursor string, batchSize int) (*PullResult, error) {
	if !c.loaded {
		if err := c.loadRecords(); err != nil {
			return nil, fmt.Errorf("failed to load records from folder: %w", err)
		}
		c.loaded = true
	}

	startIdx := 0
	if cursor != "" {
		if _, err := fmt.Sscanf(cursor, "%d", &startIdx); err != nil {
			return nil, fmt.Errorf("invalid cursor format: %w", err)
		}
	}

	if batchSize <= 0 {
		batchSize = 1000
	}

	endIdx := startIdx + batchSize
	if endIdx > len(c.records) {
		endIdx = len(c.records)
	}

	var batch []map[string]interface{}
	if startIdx < len(c.records) {
		batch = c.records[startIdx:endIdx]
	}

	hasMore := endIdx < len(c.records)
	nextCursor := ""
	if hasMore {
		nextCursor = fmt.Sprintf("%d", endIdx)
	}

	return &PullResult{
		Records:    batch,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (c *FolderConnector) AcknowledgeCheckpoint(cursor string) error {
	c.lastCheckpoint = cursor
	return nil
}

// loadRecords scans the directory for matching files and loads their contents.
func (c *FolderConnector) loadRecords() error {
	matches, err := filepath.Glob(filepath.Join(c.path, c.filePattern))
	if err != nil {
		return fmt.Errorf("failed to glob files: %w", err)
	}

	c.records = make([]map[string]interface{}, 0)

	for _, filePath := range matches {
		ext := strings.ToLower(filepath.Ext(filePath))
		switch ext {
		case ".csv":
			records, err := readCSVFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read CSV file %s: %w", filePath, err)
			}
			c.records = append(c.records, records...)
		case ".xlsx":
			records, err := readXLSXFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read XLSX file %s: %w", filePath, err)
			}
			c.records = append(c.records, records...)
		default:
			// Skip unsupported file types silently.
		}
	}

	return nil
}

// readCSVFile reads a CSV file and returns records as maps keyed by header names.
func readCSVFile(filePath string) ([]map[string]interface{}, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	var records []map[string]interface{}
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row: %w", err)
		}

		record := make(map[string]interface{}, len(headers))
		for i, header := range headers {
			if i < len(row) {
				record[strings.TrimSpace(header)] = row[i]
			} else {
				record[strings.TrimSpace(header)] = ""
			}
		}
		record["_source_file"] = filepath.Base(filePath)
		records = append(records, record)
	}

	return records, nil
}

// readXLSXFile reads the first sheet of an XLSX file and returns records.
func readXLSXFile(filePath string) ([]map[string]interface{}, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open XLSX file: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("no sheets found in XLSX file")
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read XLSX rows: %w", err)
	}

	if len(rows) < 1 {
		return nil, nil
	}

	headers := rows[0]
	var records []map[string]interface{}

	for _, row := range rows[1:] {
		record := make(map[string]interface{}, len(headers))
		for i, header := range headers {
			if i < len(row) {
				record[strings.TrimSpace(header)] = row[i]
			} else {
				record[strings.TrimSpace(header)] = ""
			}
		}
		record["_source_file"] = filepath.Base(filePath)
		records = append(records, record)
	}

	return records, nil
}

// ---------------------------------------------------------------------------
// ShareConnector: reads from a network share path (simulated as local path for LAN)
// ---------------------------------------------------------------------------

// ShareConnector reads data from a network share, treated as a local path for LAN.
type ShareConnector struct {
	folder *FolderConnector
	host   string
	share  string
}

// NewShareConnector creates a ShareConnector from config.
// Required config keys: "path" (string, UNC or local path), optional: "host", "share", "file_pattern".
func NewShareConnector(config map[string]interface{}) (*ShareConnector, error) {
	path, ok := config["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("network share connector requires 'path' in configuration")
	}

	host, _ := config["host"].(string)
	share, _ := config["share"].(string)

	folder, err := NewFolderConnector(config)
	if err != nil {
		return nil, err
	}

	return &ShareConnector{
		folder: folder,
		host:   host,
		share:  share,
	}, nil
}

func (c *ShareConnector) Type() string {
	return ConnectorShare
}

func (c *ShareConnector) Capabilities() []string {
	return []string{"pull", "incremental", "backfill", "csv", "xlsx", "network_share"}
}

func (c *ShareConnector) ValidateConfig(config map[string]interface{}) error {
	path, ok := config["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("'path' is required and must be a non-empty string")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("network share path does not exist or is inaccessible: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("network share path must be a directory")
	}
	return nil
}

func (c *ShareConnector) HealthCheck() (*HealthResult, error) {
	result, err := c.folder.HealthCheck()
	if err != nil {
		return nil, err
	}
	if result.Healthy {
		result.Message = fmt.Sprintf("network share accessible: host=%s share=%s path=%s", c.host, c.share, c.folder.path)
	}
	return result, nil
}

func (c *ShareConnector) Pull(cursor string, batchSize int) (*PullResult, error) {
	return c.folder.Pull(cursor, batchSize)
}

func (c *ShareConnector) AcknowledgeCheckpoint(cursor string) error {
	return c.folder.AcknowledgeCheckpoint(cursor)
}

// ---------------------------------------------------------------------------
// DatabaseConnector: reads from a configured database source (stub that validates config)
// ---------------------------------------------------------------------------

// DatabaseConnector connects to a MySQL/database source to pull records.
type DatabaseConnector struct {
	host      string
	port      int
	dbName    string
	user      string
	password  string
	driver    string
	table     string
	query     string
	cursorCol string
}

// NewDatabaseConnector creates a DatabaseConnector from config.
// Required config keys: "host", "db_name", "user". Optional: "port", "password",
// "driver" (default "mysql"), "table", "query", "cursor_column" (default "id").
func NewDatabaseConnector(config map[string]interface{}) (*DatabaseConnector, error) {
	host, ok := config["host"].(string)
	if !ok || host == "" {
		return nil, fmt.Errorf("database connector requires 'host' in configuration")
	}

	dbName, ok := config["db_name"].(string)
	if !ok || dbName == "" {
		return nil, fmt.Errorf("database connector requires 'db_name' in configuration")
	}

	user, ok := config["user"].(string)
	if !ok || user == "" {
		return nil, fmt.Errorf("database connector requires 'user' in configuration")
	}

	port := 3306
	if p, ok := config["port"].(float64); ok {
		port = int(p)
	}

	password, _ := config["password"].(string)

	driver := "mysql"
	if d, ok := config["driver"].(string); ok && d != "" {
		driver = d
	}

	table, _ := config["table"].(string)
	query, _ := config["query"].(string)
	cursorCol := "id"
	if cc, ok := config["cursor_column"].(string); ok && cc != "" {
		cursorCol = cc
	}

	if table == "" && query == "" {
		return nil, fmt.Errorf("database connector requires either 'table' or 'query' in configuration")
	}

	return &DatabaseConnector{
		host:      host,
		port:      port,
		dbName:    dbName,
		user:      user,
		password:  password,
		driver:    driver,
		table:     table,
		query:     query,
		cursorCol: cursorCol,
	}, nil
}

func (c *DatabaseConnector) Type() string {
	return ConnectorDB
}

func (c *DatabaseConnector) Capabilities() []string {
	return []string{"pull", "incremental", "backfill", "database", "sql"}
}

func (c *DatabaseConnector) ValidateConfig(config map[string]interface{}) error {
	required := []string{"host", "db_name", "user"}
	for _, key := range required {
		val, ok := config[key].(string)
		if !ok || val == "" {
			return fmt.Errorf("'%s' is required and must be a non-empty string", key)
		}
	}

	table, _ := config["table"].(string)
	query, _ := config["query"].(string)
	if table == "" && query == "" {
		return fmt.Errorf("either 'table' or 'query' must be provided")
	}

	// Validate real connectivity.
	host, _ := config["host"].(string)
	dbName, _ := config["db_name"].(string)
	user, _ := config["user"].(string)
	password, _ := config["password"].(string)
	port := 3306
	if p, ok := config["port"].(float64); ok {
		port = int(p)
	}
	driver := "mysql"
	if d, ok := config["driver"].(string); ok && d != "" {
		driver = d
	}

	dsn := buildDSN(driver, user, password, host, port, dbName)
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return fmt.Errorf("database connection validation failed: %w", err)
	}
	return nil
}

func (c *DatabaseConnector) HealthCheck() (*HealthResult, error) {
	now := time.Now()
	if c.host == "" || c.dbName == "" {
		return &HealthResult{
			Healthy:   false,
			Message:   "database connector has incomplete configuration",
			CheckedAt: now,
		}, nil
	}

	dsn := buildDSN(c.driver, c.user, c.password, c.host, c.port, c.dbName)
	db, err := sql.Open(c.driver, dsn)
	if err != nil {
		return &HealthResult{
			Healthy:   false,
			Message:   fmt.Sprintf("failed to open connection: %v", err),
			CheckedAt: now,
		}, nil
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return &HealthResult{
			Healthy:   false,
			Message:   fmt.Sprintf("ping failed: %v", err),
			CheckedAt: now,
		}, nil
	}

	return &HealthResult{
		Healthy:   true,
		Message:   fmt.Sprintf("database reachable: %s@%s:%d/%s", c.user, c.host, c.port, c.dbName),
		CheckedAt: now,
	}, nil
}

// Pull fetches records from the configured database source using cursor-based
// incremental pagination. An empty cursor triggers a backfill (full table scan).
// The cursor value represents the last seen value of the cursor column.
func (c *DatabaseConnector) Pull(cursor string, batchSize int) (*PullResult, error) {
	if batchSize <= 0 {
		batchSize = 1000
	}

	dsn := buildDSN(c.driver, c.user, c.password, c.host, c.port, c.dbName)
	db, err := sql.Open(c.driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}
	defer db.Close()

	// Build the query.
	var queryStr string
	if c.query != "" {
		// Custom query mode: append cursor and limit clauses.
		if cursor != "" {
			queryStr = fmt.Sprintf("SELECT * FROM (%s) AS _q WHERE %s > ? ORDER BY %s ASC LIMIT ?",
				c.query, c.cursorCol, c.cursorCol)
		} else {
			queryStr = fmt.Sprintf("SELECT * FROM (%s) AS _q ORDER BY %s ASC LIMIT ?",
				c.query, c.cursorCol)
		}
	} else {
		// Table mode.
		if cursor != "" {
			queryStr = fmt.Sprintf("SELECT * FROM %s WHERE %s > ? ORDER BY %s ASC LIMIT ?",
				c.table, c.cursorCol, c.cursorCol)
		} else {
			queryStr = fmt.Sprintf("SELECT * FROM %s ORDER BY %s ASC LIMIT ?",
				c.table, c.cursorCol)
		}
	}

	// Execute the query.
	var rows *sql.Rows
	// Request one extra row to detect hasMore.
	fetchLimit := batchSize + 1
	if cursor != "" {
		rows, err = db.Query(queryStr, cursor, fetchLimit)
	} else {
		rows, err = db.Query(queryStr, fetchLimit)
	}
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to read columns: %w", err)
	}

	var records []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		record := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			val := values[i]
			// Convert []byte to string for readability.
			if b, ok := val.([]byte); ok {
				record[col] = string(b)
			} else {
				record[col] = val
			}
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	hasMore := len(records) > batchSize
	if hasMore {
		records = records[:batchSize]
	}

	nextCursor := ""
	if len(records) > 0 {
		lastRecord := records[len(records)-1]
		if cv, ok := lastRecord[c.cursorCol]; ok {
			nextCursor = fmt.Sprintf("%v", cv)
		}
	}

	if records == nil {
		records = []map[string]interface{}{}
	}

	return &PullResult{
		Records:    records,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (c *DatabaseConnector) AcknowledgeCheckpoint(cursor string) error {
	return nil
}

// buildDSN constructs a data source name for the given driver.
func buildDSN(driver, user, password, host string, port int, dbName string) string {
	switch driver {
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			user, password, host, port, dbName)
	default:
		// Fallback generic DSN.
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, dbName)
	}
}
