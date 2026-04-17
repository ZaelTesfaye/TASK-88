//go:build integration

package api

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/models"
	"backend/internal/router"
)

func timeNow() time.Time { return time.Now() }

var testDB *gorm.DB

// getTestDB returns a real GORM database connection for integration tests.
func getTestDB() *gorm.DB {
	if testDB != nil {
		return testDB
	}

	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		return nil
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration test DB connection failed: %v\n", err)
		return nil
	}

	if err := database.AutoMigrate(db); err != nil {
		fmt.Fprintf(os.Stderr, "integration test DB migration failed: %v\n", err)
		return nil
	}

	testDB = db
	return testDB
}

// realRouter builds the production router backed by a real database.
func realRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	cfg := config.GetConfig()
	return router.SetupRouter(cfg, db)
}

// cleanTestData truncates all tables between tests.
func cleanTestData(db *gorm.DB) {
	tables := []string{
		"purge_runs", "legal_holds", "retention_policies",
		"password_reset_requests", "key_rings", "sensitive_field_registry",
		"integration_deliveries", "integration_endpoints", "connector_definitions",
		"ingestion_checkpoints", "ingestion_failures", "ingestion_jobs",
		"import_sources", "master_version_items", "master_versions",
		"deactivation_events", "master_records", "context_assignments",
		"sessions", "audit_logs", "audit_delete_requests", "report_runs",
		"report_schedules", "analytics_kpi_definitions", "media_assets",
		"org_nodes", "users",
	}
	for _, t := range tables {
		db.Exec("DELETE FROM " + t)
	}
	db.Exec("ALTER TABLE users AUTO_INCREMENT = 1")
	db.Exec("ALTER TABLE org_nodes AUTO_INCREMENT = 1")
	db.Exec("ALTER TABLE master_records AUTO_INCREMENT = 1")
	db.Exec("ALTER TABLE master_versions AUTO_INCREMENT = 1")
	db.Exec("ALTER TABLE import_sources AUTO_INCREMENT = 1")
	db.Exec("ALTER TABLE ingestion_jobs AUTO_INCREMENT = 1")
	db.Exec("ALTER TABLE media_assets AUTO_INCREMENT = 1")
	db.Exec("ALTER TABLE report_schedules AUTO_INCREMENT = 1")
	db.Exec("ALTER TABLE report_runs AUTO_INCREMENT = 1")
}

// loginAndGetToken creates a user with a real password, logs in via the real
// auth flow, and returns the access token. This is the ONLY way to obtain a
// token in no-mock tests — no fakeAuthMiddleware or signToken.
func loginAndGetToken(db *gorm.DB, r *gin.Engine, username, role, city, dept string) string {
	hash, _ := auth.HashPassword("TestPass123!")
	user := models.User{
		Username: username, PasswordHash: hash, Role: role,
		CityScope: city, DepartmentScope: dept, Status: "active",
	}
	db.Create(&user)
	w := doRequest(r, "POST", "/api/v1/auth/login", "", map[string]interface{}{
		"username": username, "password": "TestPass123!",
	})
	body := parseBody(w)
	tok, _ := body["token"].(string)
	return tok
}
