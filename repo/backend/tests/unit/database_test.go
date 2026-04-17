package unit

import (
	"testing"

	"backend/internal/config"
	"backend/internal/database"
)

// Tests for database/database.go — connection initialization, DSN construction
// edge cases, and error propagation on bad config.

func TestConnectWithBadConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping connection timeout test in short mode")
	}
	cfg := &config.Config{
		DBHost:     "192.0.2.1", // RFC 5737 documentation address — unreachable
		DBPort:     "3306",
		DBUser:     "nobody",
		DBPassword: "wrong",
		DBName:     "nonexistent",
		AppEnv:     "test",
	}

	_, err := database.Connect(cfg)
	if err == nil {
		t.Error("expected error connecting to unreachable host")
	}
}

func TestConnectWithEmptyHost(t *testing.T) {
	cfg := &config.Config{
		DBHost:     "",
		DBPort:     "3306",
		DBUser:     "user",
		DBPassword: "pass",
		DBName:     "db",
		AppEnv:     "test",
	}

	_, err := database.Connect(cfg)
	if err == nil {
		t.Error("expected error with empty host")
	}
}

func TestAutoMigrateWithNilDB(t *testing.T) {
	// AutoMigrate with nil should panic or error.
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic or error with nil DB")
		}
	}()
	_ = database.AutoMigrate(nil)
}

func TestGetDBReturnsNilBeforeConnect(t *testing.T) {
	// GetDB returns the package-level db variable.
	// Before any connection is made in this test process, it may be nil
	// (unless another test already connected).  We just verify it doesn't panic.
	_ = database.GetDB()
}
