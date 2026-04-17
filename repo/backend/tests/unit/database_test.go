package unit

import (
	"testing"

	"backend/internal/config"
	"backend/internal/database"
)

// Tests for database/database.go — connection initialization, DSN construction
// edge cases, and error propagation on bad config.

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

func TestConnectWithInvalidPort(t *testing.T) {
	// Port "0" produces an invalid DSN that fails at parse/dial stage
	// quickly without waiting for a network timeout.
	cfg := &config.Config{
		DBHost:     "127.0.0.1",
		DBPort:     "0",
		DBUser:     "nobody",
		DBPassword: "wrong",
		DBName:     "nonexistent",
		AppEnv:     "test",
	}

	_, err := database.Connect(cfg)
	if err == nil {
		t.Error("expected error with invalid port 0")
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

func TestGetDBReturnsWithoutPanic(t *testing.T) {
	// GetDB returns the package-level db variable.
	// Before any connection is made in this test process, it may be nil
	// (unless another test already connected). We just verify it doesn't panic.
	_ = database.GetDB()
}
