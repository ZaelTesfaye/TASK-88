package api

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/models"
	"backend/internal/router"
)

func timeNow() time.Time { return time.Now() }

// testDB holds a shared database connection for integration tests that need a
// real database. It is lazily initialised by getTestDB(). If the TEST_DB_DSN
// environment variable is not set, the helper returns nil and the caller should
// call t.Skip("TEST_DB_DSN not set").
var testDB *gorm.DB

// getTestDB returns a real GORM database connection for integration tests.
// Set TEST_DB_DSN (e.g. "root:pass@tcp(127.0.0.1:3306)/hub_test?charset=utf8mb4&parseTime=True")
// to enable database-backed tests.
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

	// Auto-migrate all models.
	if err := database.AutoMigrate(db); err != nil {
		fmt.Fprintf(os.Stderr, "integration test DB migration failed: %v\n", err)
		return nil
	}

	testDB = db
	return testDB
}

// realRouter builds the production router.SetupRouter() backed by the given
// database. This exercises all real middleware, RBAC, auth, and handlers.
func realRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	cfg := config.GetConfig()
	return router.SetupRouter(cfg, db)
}

// seedTestUser inserts a user into the test database and returns its ID.
func seedTestUser(db *gorm.DB, username, role, city, dept string) uint {
	user := models.User{
		Username:        username,
		PasswordHash:    "$argon2id$v=19$m=65536,t=1,p=4$dGVzdHNhbHQ$testhash_placeholder",
		Role:            role,
		CityScope:       city,
		DepartmentScope: dept,
		Status:          "active",
	}
	db.Create(&user)
	return user.ID
}

// seedSession inserts a session for the user so that AuthRequired middleware
// can look it up. Returns the JTI.
func seedSession(db *gorm.DB, userID uint, jti string) {
	session := models.Session{
		UserID:         userID,
		JwtJTI:         jti,
		IssuedAt:       timeNow(),
		LastActivityAt: timeNow(),
		ExpiresAt:      timeNow().Add(30 * 60 * 1e9), // 30 min
		IPAddress:      "127.0.0.1",
		UserAgent:      "test-agent",
	}
	db.Create(&session)
}

// cleanTestData truncates key tables between tests.
func cleanTestData(db *gorm.DB) {
	tables := []string{
		"ingestion_checkpoints", "ingestion_failures", "ingestion_jobs",
		"master_version_items", "master_versions", "deactivation_events",
		"master_records", "context_assignments", "sessions",
		"audit_logs", "audit_delete_requests", "report_runs",
		"report_schedules", "org_nodes", "users",
	}
	for _, t := range tables {
		db.Exec("DELETE FROM " + t)
	}
	db.Exec("ALTER TABLE users AUTO_INCREMENT = 1")
	db.Exec("ALTER TABLE org_nodes AUTO_INCREMENT = 1")
	db.Exec("ALTER TABLE master_records AUTO_INCREMENT = 1")
}
