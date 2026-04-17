package unit

import (
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"backend/internal/database"
	"backend/internal/models"
	"backend/internal/org"
	"backend/internal/rbac"
)

// getUnitTestDB returns a GORM DB for unit/integration tests.
var unitTestDB *gorm.DB

func getUnitTestDB() *gorm.DB {
	if unitTestDB != nil {
		return unitTestDB
	}
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		return nil
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil
	}
	if err := database.AutoMigrate(db); err != nil {
		return nil
	}
	unitTestDB = db
	return unitTestDB
}

func cleanUnitTestData(db *gorm.DB) {
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
}

// ---------- OrgService tests ----------

func TestOrgServiceCreateNode(t *testing.T) {
	db := getUnitTestDB()
	if db == nil {
		t.Skip("TEST_DB_DSN not set — skipping integration test")
	}
	cleanUnitTestData(db)

	svc := org.NewOrgService(db)
	node, err := svc.CreateNode(org.CreateNodeRequest{
		LevelCode:  "company",
		LevelLabel: "Company",
		Name:       "TestCorp",
	})
	if err != nil {
		t.Fatalf("CreateNode failed: %v", err)
	}
	if node.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if node.Name != "TestCorp" {
		t.Errorf("expected name 'TestCorp', got %q", node.Name)
	}
	if !node.IsActive {
		t.Error("expected IsActive to be true")
	}
}

func TestOrgServiceCreateNodeInvalidLevelCode(t *testing.T) {
	db := getUnitTestDB()
	if db == nil {
		t.Skip("TEST_DB_DSN not set — skipping integration test")
	}
	cleanUnitTestData(db)

	svc := org.NewOrgService(db)
	_, err := svc.CreateNode(org.CreateNodeRequest{
		LevelCode:  "invalid_code",
		LevelLabel: "Bad Level",
		Name:       "BadNode",
	})
	if err == nil {
		t.Fatal("expected error for invalid level_code")
	}
}

func TestOrgServiceGetTree(t *testing.T) {
	db := getUnitTestDB()
	if db == nil {
		t.Skip("TEST_DB_DSN not set — skipping integration test")
	}
	cleanUnitTestData(db)

	svc := org.NewOrgService(db)
	svc.CreateNode(org.CreateNodeRequest{
		LevelCode: "company", LevelLabel: "Company", Name: "Corp",
	})

	tree, err := svc.GetTree()
	if err != nil {
		t.Fatalf("GetTree failed: %v", err)
	}
	if len(tree) == 0 {
		t.Error("expected at least one tree node")
	}
	if tree[0].Name != "Corp" {
		t.Errorf("expected root name 'Corp', got %q", tree[0].Name)
	}
}

func TestOrgServiceDeleteNode(t *testing.T) {
	db := getUnitTestDB()
	if db == nil {
		t.Skip("TEST_DB_DSN not set — skipping integration test")
	}
	cleanUnitTestData(db)

	svc := org.NewOrgService(db)
	node, _ := svc.CreateNode(org.CreateNodeRequest{
		LevelCode: "company", LevelLabel: "Company", Name: "ToDelete",
	})

	err := svc.DeleteNode(node.ID)
	if err != nil {
		t.Fatalf("DeleteNode failed: %v", err)
	}

	// Verify it's gone.
	var count int64
	db.Model(&models.OrgNode{}).Where("id = ?", node.ID).Count(&count)
	if count != 0 {
		t.Error("expected node to be deleted")
	}
}

func TestOrgServiceDeleteNodeWithChildren(t *testing.T) {
	db := getUnitTestDB()
	if db == nil {
		t.Skip("TEST_DB_DSN not set — skipping integration test")
	}
	cleanUnitTestData(db)

	svc := org.NewOrgService(db)
	parent, _ := svc.CreateNode(org.CreateNodeRequest{
		LevelCode: "company", LevelLabel: "Company", Name: "Parent",
	})
	svc.CreateNode(org.CreateNodeRequest{
		ParentID: &parent.ID, LevelCode: "city", LevelLabel: "City", Name: "Child",
	})

	err := svc.DeleteNode(parent.ID)
	if err == nil {
		t.Fatal("expected error when deleting node with active children")
	}
}

func TestOrgServiceSwitchContext(t *testing.T) {
	db := getUnitTestDB()
	if db == nil {
		t.Skip("TEST_DB_DSN not set — skipping integration test")
	}
	cleanUnitTestData(db)

	// Seed user and node.
	user := models.User{
		Username: "ctxuser", PasswordHash: "x", Role: rbac.SystemAdmin,
		CityScope: "*", DepartmentScope: "*", Status: "active",
	}
	db.Create(&user)

	svc := org.NewOrgService(db)
	node, _ := svc.CreateNode(org.CreateNodeRequest{
		LevelCode: "company", LevelLabel: "Company", Name: "CtxNode",
	})

	err := svc.SwitchContext(user.ID, node.ID)
	if err != nil {
		t.Fatalf("SwitchContext failed: %v", err)
	}

	// Verify context.
	ctxNode, scopeIDs, err := svc.GetUserContext(user.ID)
	if err != nil {
		t.Fatalf("GetUserContext failed: %v", err)
	}
	if ctxNode == nil {
		t.Fatal("expected context node")
	}
	if ctxNode.ID != node.ID {
		t.Errorf("expected context node ID %d, got %d", node.ID, ctxNode.ID)
	}
	if len(scopeIDs) == 0 {
		t.Error("expected at least one scope ID")
	}
}

func TestOrgServiceValidateNoCycle(t *testing.T) {
	db := getUnitTestDB()
	if db == nil {
		t.Skip("TEST_DB_DSN not set — skipping integration test")
	}
	cleanUnitTestData(db)

	svc := org.NewOrgService(db)
	parent, _ := svc.CreateNode(org.CreateNodeRequest{
		LevelCode: "company", LevelLabel: "Company", Name: "Parent",
	})
	child, _ := svc.CreateNode(org.CreateNodeRequest{
		ParentID: &parent.ID, LevelCode: "city", LevelLabel: "City", Name: "Child",
	})

	// Trying to set parent's parent to child should fail (cycle).
	err := svc.ValidateNoCycle(parent.ID, child.ID)
	if err == nil {
		t.Error("expected cycle validation error")
	}
}

func TestOrgServiceGetBreadcrumb(t *testing.T) {
	db := getUnitTestDB()
	if db == nil {
		t.Skip("TEST_DB_DSN not set — skipping integration test")
	}
	cleanUnitTestData(db)

	svc := org.NewOrgService(db)
	parent, _ := svc.CreateNode(org.CreateNodeRequest{
		LevelCode: "company", LevelLabel: "Company", Name: "Root",
	})
	child, _ := svc.CreateNode(org.CreateNodeRequest{
		ParentID: &parent.ID, LevelCode: "city", LevelLabel: "City", Name: "Branch",
	})

	bc, err := svc.GetBreadcrumb(child.ID)
	if err != nil {
		t.Fatalf("GetBreadcrumb failed: %v", err)
	}
	if len(bc) != 2 {
		t.Fatalf("expected 2 breadcrumb items, got %d", len(bc))
	}
	if bc[0].Name != "Root" {
		t.Errorf("first breadcrumb should be Root, got %q", bc[0].Name)
	}
	if bc[1].Name != "Branch" {
		t.Errorf("second breadcrumb should be Branch, got %q", bc[1].Name)
	}
}
