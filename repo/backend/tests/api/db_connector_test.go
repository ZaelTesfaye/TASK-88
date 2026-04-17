//go:build integration

package api

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"backend/internal/ingestion"
)

// --- 9.3 DB connector integration tests ---

// getConnectorTestDSN returns the test database DSN for connector tests.
// Set TEST_DB_DSN to enable these tests.
func getConnectorTestDSN() string {
	return os.Getenv("TEST_DB_DSN")
}

// setupConnectorFixture creates a temporary table with test data.
func setupConnectorFixture(dsn string) error {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS connector_test_data (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		value VARCHAR(255),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return err
	}

	// Clear existing data.
	_, err = db.Exec("DELETE FROM connector_test_data")
	if err != nil {
		return err
	}

	// Seed test rows.
	for i := 1; i <= 10; i++ {
		_, err = db.Exec("INSERT INTO connector_test_data (name, value) VALUES (?, ?)",
			fmt.Sprintf("item_%d", i), fmt.Sprintf("value_%d", i))
		if err != nil {
			return err
		}
	}
	return nil
}

func teardownConnectorFixture(dsn string) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return
	}
	defer db.Close()
	db.Exec("DROP TABLE IF EXISTS connector_test_data")
}

// TestDatabaseConnectorFullPull tests that Pull with no cursor returns all rows
// (backfill mode).
func TestDatabaseConnectorFullPull(t *testing.T) {
	dsn := getConnectorTestDSN()
	if dsn == "" {
		t.Fatal("TEST_DB_DSN must be set to run no-mock API tests")
	}

	if err := setupConnectorFixture(dsn); err != nil {
		t.Fatalf("failed to set up fixture: %v", err)
	}
	defer teardownConnectorFixture(dsn)

	// Parse DSN components for config map.
	config := parseDSNToConfig(dsn)
	config["table"] = "connector_test_data"
	config["cursor_column"] = "id"

	connector, err := ingestion.NewDatabaseConnector(config)
	if err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	// Full pull (no cursor = backfill).
	result, err := connector.Pull("", 100)
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	if len(result.Records) != 10 {
		t.Errorf("expected 10 records from full pull, got %d", len(result.Records))
	}
	if result.HasMore {
		t.Error("expected HasMore=false for a pull that returned all rows")
	}
}

// TestDatabaseConnectorIncrementalPull tests that Pull with a cursor only
// returns rows newer than the checkpoint.
func TestDatabaseConnectorIncrementalPull(t *testing.T) {
	dsn := getConnectorTestDSN()
	if dsn == "" {
		t.Fatal("TEST_DB_DSN must be set to run no-mock API tests")
	}

	if err := setupConnectorFixture(dsn); err != nil {
		t.Fatalf("failed to set up fixture: %v", err)
	}
	defer teardownConnectorFixture(dsn)

	config := parseDSNToConfig(dsn)
	config["table"] = "connector_test_data"
	config["cursor_column"] = "id"

	connector, err := ingestion.NewDatabaseConnector(config)
	if err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	// Incremental pull with cursor at id=5 → should get rows 6-10.
	result, err := connector.Pull("5", 100)
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	if len(result.Records) != 5 {
		t.Errorf("expected 5 records from incremental pull (id > 5), got %d", len(result.Records))
	}
}

// TestDatabaseConnectorConnectionFailure tests that Pull returns an error
// when the database is unreachable.
func TestDatabaseConnectorConnectionFailure(t *testing.T) {
	config := map[string]interface{}{
		"host":    "192.0.2.1", // RFC 5737 documentation IP — guaranteed unreachable
		"port":    float64(3306),
		"db_name": "nonexistent",
		"user":    "nobody",
		"table":   "test",
	}

	connector, err := ingestion.NewDatabaseConnector(config)
	if err != nil {
		t.Fatalf("unexpected error creating connector: %v", err)
	}

	_, err = connector.Pull("", 10)
	if err == nil {
		t.Error("expected error for unreachable database, got nil")
	}
}

// TestDatabaseConnectorHealthCheck verifies the health check with a real DB.
func TestDatabaseConnectorHealthCheck(t *testing.T) {
	dsn := getConnectorTestDSN()
	if dsn == "" {
		t.Fatal("TEST_DB_DSN must be set to run no-mock API tests")
	}

	config := parseDSNToConfig(dsn)
	config["table"] = "connector_test_data"

	connector, err := ingestion.NewDatabaseConnector(config)
	if err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	result, err := connector.HealthCheck()
	if err != nil {
		t.Fatalf("HealthCheck error: %v", err)
	}
	if !result.Healthy {
		t.Errorf("expected healthy=true, got message: %s", result.Message)
	}
}

// parseDSNToConfig extracts a simplified config map from a MySQL DSN string
// for use with NewDatabaseConnector. This is a best-effort helper for tests.
func parseDSNToConfig(dsn string) map[string]interface{} {
	// MySQL DSN: user:pass@tcp(host:port)/dbname?params
	config := map[string]interface{}{
		"host":    "127.0.0.1",
		"port":    float64(3306),
		"db_name": "hub_test",
		"user":    "root",
	}

	// Try to parse basic components.
	// Format: user:password@tcp(host:port)/dbname
	atIdx := -1
	for i, c := range dsn {
		if c == '@' {
			atIdx = i
			break
		}
	}
	if atIdx > 0 {
		userPart := dsn[:atIdx]
		rest := dsn[atIdx+1:]

		// Parse user:password
		colonIdx := -1
		for i, c := range userPart {
			if c == ':' {
				colonIdx = i
				break
			}
		}
		if colonIdx > 0 {
			config["user"] = userPart[:colonIdx]
			config["password"] = userPart[colonIdx+1:]
		} else {
			config["user"] = userPart
		}

		// Parse tcp(host:port)/dbname
		if len(rest) > 4 && rest[:4] == "tcp(" {
			closeParen := -1
			for i, c := range rest {
				if c == ')' {
					closeParen = i
					break
				}
			}
			if closeParen > 0 {
				hostPort := rest[4:closeParen]
				hostColonIdx := -1
				for i, c := range hostPort {
					if c == ':' {
						hostColonIdx = i
						break
					}
				}
				if hostColonIdx > 0 {
					config["host"] = hostPort[:hostColonIdx]
					var port int
					fmt.Sscanf(hostPort[hostColonIdx+1:], "%d", &port)
					config["port"] = float64(port)
				} else {
					config["host"] = hostPort
				}

				// Parse /dbname?params
				dbPart := rest[closeParen+1:]
				if len(dbPart) > 0 && dbPart[0] == '/' {
					dbPart = dbPart[1:]
				}
				qIdx := -1
				for i, c := range dbPart {
					if c == '?' {
						qIdx = i
						break
					}
				}
				if qIdx > 0 {
					config["db_name"] = dbPart[:qIdx]
				} else {
					config["db_name"] = dbPart
				}
			}
		}
	}

	return config
}
