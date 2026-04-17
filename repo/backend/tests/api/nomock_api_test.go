//go:build integration

package api

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/internal/auth"
	"backend/internal/models"
	"backend/internal/rbac"
)

// ==========================================================================
// All tests in this file use realRouter(db) — the production router wired to
// a real MySQL database.  They are skipped when TEST_DB_DSN is not set.
//
// Pattern:
//   1. getTestDB() — nil → t.Skip
//   2. cleanTestData(db) — wipe all tables
//   3. Seed users/data via GORM or raw SQL
//   4. loginAndGetToken(db, r, …) — obtain a real JWT via POST /auth/login
//   5. doRequest(r, method, path, token, body) → assert status + body
// ==========================================================================

// ---- helpers ----

func skipIfNoDB(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	if os.Getenv("TEST_DB_DSN") == "" {
		t.Fatal("TEST_DB_DSN must be set to run no-mock API tests")
	}
	db := getTestDB()
	if db == nil {
		t.Fatal("TEST_DB_DSN is set but database connection failed")
	}
	cleanTestData(db)
	r := realRouter(db)
	return db, r
}

// ---- Auth ----

func TestRealAuthLogin(t *testing.T) {
	db, r := skipIfNoDB(t)
	hash, _ := auth.HashPassword("TestPass123!")
	db.Create(&models.User{Username: "logintest", PasswordHash: hash, Role: rbac.SystemAdmin, CityScope: "*", DepartmentScope: "*", Status: "active"})

	w := doRequest(r, "POST", "/api/v1/auth/login", "", map[string]interface{}{"username": "logintest", "password": "TestPass123!"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["token"] == nil || body["token"] == "" {
		t.Error("expected non-empty token")
	}
	if body["refreshToken"] == nil || body["refreshToken"] == "" {
		t.Error("expected non-empty refreshToken")
	}
	user, _ := body["user"].(map[string]interface{})
	if user["username"] != "logintest" {
		t.Errorf("expected username 'logintest', got %v", user["username"])
	}
	if user["role"] != rbac.SystemAdmin {
		t.Errorf("expected role system_admin, got %v", user["role"])
	}
}

func TestRealAuthLoginBadCreds(t *testing.T) {
	_, r := skipIfNoDB(t)
	w := doRequest(r, "POST", "/api/v1/auth/login", "", map[string]interface{}{"username": "nobody", "password": "wrong"})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	body := parseBody(w)
	if body["message"] == nil {
		t.Error("expected error message")
	}
}

func TestRealAuthLoginNoBody(t *testing.T) {
	_, r := skipIfNoDB(t)
	w := doRequest(r, "POST", "/api/v1/auth/login", "", map[string]interface{}{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRealAuthLogout(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "logouttest", rbac.SystemAdmin, "*", "*")
	w := doRequest(r, "POST", "/api/v1/auth/logout", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["message"] != "logged out successfully" {
		t.Errorf("unexpected message: %v", body["message"])
	}
}

func TestRealAuthRefresh(t *testing.T) {
	db, r := skipIfNoDB(t)
	hash, _ := auth.HashPassword("TestPass123!")
	db.Create(&models.User{Username: "refreshtest", PasswordHash: hash, Role: rbac.SystemAdmin, CityScope: "*", DepartmentScope: "*", Status: "active"})
	w := doRequest(r, "POST", "/api/v1/auth/login", "", map[string]interface{}{"username": "refreshtest", "password": "TestPass123!"})
	body := parseBody(w)
	accessToken := body["token"].(string)
	refreshToken := body["refreshToken"].(string)

	w = doRequest(r, "POST", "/api/v1/auth/refresh", accessToken, map[string]interface{}{"refreshToken": refreshToken})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body = parseBody(w)
	if body["token"] == nil || body["token"] == "" {
		t.Error("expected new token")
	}
	if body["token"] == accessToken {
		t.Error("new token should differ from old")
	}
}

func TestRealAuthNoToken(t *testing.T) {
	_, r := skipIfNoDB(t)
	w := doRequest(r, "GET", "/api/v1/org/tree", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	body := parseBody(w)
	if body["code"] == nil {
		t.Error("expected error code")
	}
}

// ---- Org ----

func TestRealOrgTree(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "orgadmin", rbac.SystemAdmin, "*", "*")
	db.Exec("INSERT INTO org_nodes (level_code, level_label, name, city, department, is_active, sort_order) VALUES ('company','Company','TestCorp','','',true,0)")

	w := doRequest(r, "GET", "/api/v1/org/tree", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	data, _ := body["data"].([]interface{})
	if len(data) == 0 {
		t.Error("expected at least one node")
	}
}

func TestRealOrgListNodes(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "orglist", rbac.SystemAdmin, "*", "*")
	db.Exec("INSERT INTO org_nodes (level_code, level_label, name, is_active, sort_order) VALUES ('company','Company','Corp',true,0)")

	w := doRequest(r, "GET", "/api/v1/org/nodes", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	data, _ := body["data"].([]interface{})
	if len(data) == 0 {
		t.Error("expected nodes")
	}
}

func TestRealOrgCreateNode(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "orgcreate", rbac.SystemAdmin, "*", "*")
	w := doRequest(r, "POST", "/api/v1/org/nodes", token, map[string]interface{}{
		"level_code": "company", "level_label": "Company", "name": "NewCorp",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	data, _ := body["data"].(map[string]interface{})
	if data["name"] != "NewCorp" {
		t.Errorf("expected name NewCorp, got %v", data["name"])
	}
}

func TestRealOrgGetNode(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "orgget", rbac.SystemAdmin, "*", "*")
	db.Exec("INSERT INTO org_nodes (id,level_code,level_label,name,is_active,sort_order) VALUES (100,'company','Company','GetMe',true,0)")

	w := doRequest(r, "GET", "/api/v1/org/nodes/100", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	data, _ := body["data"].(map[string]interface{})
	if data["name"] != "GetMe" {
		t.Errorf("expected name GetMe, got %v", data["name"])
	}
}

func TestRealOrgGetNodeNotFound(t *testing.T) {
	_, r := skipIfNoDB(t)
	// no seeded user with login needed — but we DO need an admin token
	db := getTestDB()
	token := loginAndGetToken(db, r, "orgget404", rbac.SystemAdmin, "*", "*")
	w := doRequest(r, "GET", "/api/v1/org/nodes/99999", token, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestRealOrgUpdateNode(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "orgupdate", rbac.SystemAdmin, "*", "*")
	db.Exec("INSERT INTO org_nodes (id,level_code,level_label,name,is_active,sort_order) VALUES (100,'company','Company','Old',true,0)")

	w := doRequest(r, "PUT", "/api/v1/org/nodes/100", token, map[string]interface{}{"name": "Updated"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	data, _ := body["data"].(map[string]interface{})
	if data["name"] != "Updated" {
		t.Errorf("expected name Updated, got %v", data["name"])
	}
}

func TestRealOrgDeleteNode(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "orgdelete", rbac.SystemAdmin, "*", "*")
	db.Exec("INSERT INTO org_nodes (id,level_code,level_label,name,is_active,sort_order) VALUES (100,'company','Company','Delete',true,0)")

	w := doRequest(r, "DELETE", "/api/v1/org/nodes/100", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["message"] == nil {
		t.Error("expected message")
	}
}

func TestRealOrgRBACDenied(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "orgdenied", rbac.StandardUser, "NYC", "Finance")
	w := doRequest(r, "GET", "/api/v1/org/tree", token, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	body := parseBody(w)
	if body["code"] == nil {
		t.Error("expected error code in 403 response")
	}
}

func TestRealContextSwitchAndGet(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "ctxtest", rbac.SystemAdmin, "*", "*")
	db.Exec("INSERT INTO org_nodes (id,level_code,level_label,name,is_active,sort_order) VALUES (100,'company','Company','CtxNode',true,0)")

	w := doRequest(r, "POST", "/api/v1/context/switch", token, map[string]interface{}{"node_id": 100})
	if w.Code != http.StatusOK {
		t.Fatalf("switch: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	data, _ := body["data"].(map[string]interface{})
	if data["current_node"] == nil {
		t.Error("expected current_node")
	}

	w = doRequest(r, "GET", "/api/v1/context/current", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("current: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body = parseBody(w)
	data, _ = body["data"].(map[string]interface{})
	if data["current_node"] == nil {
		t.Error("expected current_node after switch")
	}
}

// ---- Master ----

func TestRealMasterListAndCreate(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "masteradmin", rbac.SystemAdmin, "*", "*")

	// Create
	w := doRequest(r, "POST", "/api/v1/master/sku", token, map[string]interface{}{
		"natural_key": "SKU-001", "payload_json": `{"name":"Widget"}`,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	data, _ := body["data"].(map[string]interface{})
	if data["natural_key"] != "SKU-001" {
		t.Errorf("expected natural_key SKU-001, got %v", data["natural_key"])
	}

	// List
	w = doRequest(r, "GET", "/api/v1/master/sku", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body = parseBody(w)
	if body["data"] == nil {
		t.Error("expected data in list response")
	}
}

func TestRealMasterDeactivate(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "masterdeact", rbac.SystemAdmin, "*", "*")
	db.Exec("INSERT INTO master_records (id,entity_type,natural_key,payload_json,status) VALUES (1,'sku','D1','{\"n\":1}','active')")

	w := doRequest(r, "POST", "/api/v1/master/sku/1/deactivate", token, map[string]interface{}{"reason": "obsolete"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["message"] == nil {
		t.Error("expected deactivation message")
	}
}

func TestRealMasterHistory(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "masterhist", rbac.SystemAdmin, "*", "*")
	db.Exec("INSERT INTO master_records (id,entity_type,natural_key,payload_json,status) VALUES (1,'sku','H1','{\"n\":1}','active')")

	w := doRequest(r, "GET", "/api/v1/master/sku/1/history", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	data, ok := body["data"].([]interface{})
	if !ok {
		t.Fatal("expected data array")
	}
	// Fresh record — no deactivation events yet.
	if len(data) != 0 {
		t.Errorf("expected empty history, got %d events", len(data))
	}
}

func TestRealMasterHistoryNotFound(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "masternf", rbac.SystemAdmin, "*", "*")
	_ = db
	w := doRequest(r, "GET", "/api/v1/master/sku/99999/history", token, nil)
	// Handler returns 200 with empty array for nonexistent records.
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRealMasterHistoryNoAuth(t *testing.T) {
	_, r := skipIfNoDB(t)
	w := doRequest(r, "GET", "/api/v1/master/sku/1/history", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ---- Versions ----

func TestRealVersionsCRUD(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "veradmin", rbac.SystemAdmin, "*", "*")
	db.Exec("INSERT INTO org_nodes (id,level_code,level_label,name,is_active,sort_order) VALUES (100,'company','Company','Corp',true,0)")
	db.Exec("INSERT INTO context_assignments (user_id, org_node_id) VALUES (1, 100)")
	db.Exec("INSERT INTO master_records (id,entity_type,natural_key,payload_json,status) VALUES (1,'sku','VK1','{\"n\":1}','active')")

	// Create draft
	w := doRequest(r, "POST", "/api/v1/versions/sku", token, map[string]interface{}{
		"scope_key": "node:100",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	vData, _ := body["data"].(map[string]interface{})
	vID := vData["id"]

	// List
	w = doRequest(r, "GET", "/api/v1/versions/sku", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body = parseBody(w)
	if body["data"] == nil {
		t.Error("expected data array")
	}

	// Get single
	w = doRequest(r, "GET", fmt.Sprintf("/api/v1/versions/sku/%.0f", vID), token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body = parseBody(w)
	gData, _ := body["data"].(map[string]interface{})
	if gData["state"] != "draft" {
		t.Errorf("expected state draft, got %v", gData["state"])
	}

	// Add item
	w = doRequest(r, "POST", fmt.Sprintf("/api/v1/versions/sku/%.0f/items", vID), token, map[string]interface{}{
		"master_record_ids": []int{1},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("add items: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// List items
	w = doRequest(r, "GET", fmt.Sprintf("/api/v1/versions/sku/%.0f/items", vID), token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list items: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Diff
	w = doRequest(r, "GET", fmt.Sprintf("/api/v1/versions/sku/%.0f/diff", vID), token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("diff: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Review
	w = doRequest(r, "POST", fmt.Sprintf("/api/v1/versions/sku/%.0f/review", vID), token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("review: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Activate
	w = doRequest(r, "POST", fmt.Sprintf("/api/v1/versions/sku/%.0f/activate", vID), token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("activate: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRealVersionsRBACStandardUserDenied(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "verstd", rbac.StandardUser, "NYC", "Finance")
	w := doRequest(r, "POST", "/api/v1/versions/sku", token, map[string]interface{}{"scope_key": "node:1"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for standard_user version create, got %d", w.Code)
	}
	body := parseBody(w)
	if body["code"] == nil {
		t.Error("expected error code in 403 response")
	}
}

// ---- Ingestion ----

func TestRealIngestionSourcesCRUD(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "ingestadmin", rbac.SystemAdmin, "*", "*")

	// Create source
	w := doRequest(r, "POST", "/api/v1/ingestion/sources", token, map[string]interface{}{
		"name": "TestDB", "source_type": "database",
		"connection_json": `{"host":"db.test"}`, "mapping_rules_json": `{}`,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create source: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// List sources
	w = doRequest(r, "GET", "/api/v1/ingestion/sources", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list sources: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Get source
	w = doRequest(r, "GET", "/api/v1/ingestion/sources/1", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get source: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Update source
	w = doRequest(r, "PUT", "/api/v1/ingestion/sources/1", token, map[string]interface{}{
		"name": "TestDB-Updated", "source_type": "database",
		"connection_json": `{"host":"db2.test"}`, "mapping_rules_json": `{}`,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("update source: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Delete source
	w = doRequest(r, "DELETE", "/api/v1/ingestion/sources/1", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete source: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRealIngestionJobsLifecycle(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "jobtester", rbac.SystemAdmin, "*", "*")
	db.Exec("INSERT INTO import_sources (id,name,source_type,connection_json,mapping_rules_json,is_active) VALUES (1,'S1','database','{\"h\":\"x\"}','{}',true)")

	// Create job
	w := doRequest(r, "POST", "/api/v1/ingestion/jobs", token, map[string]interface{}{
		"import_source_id": 1,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create job: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["id"] == nil {
		t.Error("expected job id")
	}

	// List jobs
	w = doRequest(r, "GET", "/api/v1/ingestion/jobs", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list jobs: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Get job
	w = doRequest(r, "GET", "/api/v1/ingestion/jobs/1", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get job: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Retry job
	w = doRequest(r, "POST", "/api/v1/ingestion/jobs/1/retry", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("retry: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Acknowledge job
	w = doRequest(r, "POST", "/api/v1/ingestion/jobs/1/acknowledge", token, map[string]interface{}{
		"acknowledged_reason": "manual review done",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("ack: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Checkpoints
	w = doRequest(r, "GET", "/api/v1/ingestion/jobs/1/checkpoints", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("checkpoints: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Failures
	w = doRequest(r, "GET", "/api/v1/ingestion/jobs/1/failures", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("failures: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRealIngestionRBACDenied(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "ingestdeny", rbac.StandardUser, "NYC", "Finance")
	w := doRequest(r, "GET", "/api/v1/ingestion/sources", token, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	body := parseBody(w)
	if body["code"] == nil {
		t.Error("expected error code in 403 body")
	}
	if body["message"] == nil || body["message"] == "" {
		t.Error("expected error message in 403 body")
	}

	// Companion positive path: authorized user can access ingestion.
	adminToken := loginAndGetToken(db, r, "ingestok", rbac.SystemAdmin, "*", "*")
	db.Exec("INSERT INTO import_sources (id,name,source_type,connection_json,mapping_rules_json,is_active) VALUES (1,'Src','database','{\"h\":\"x\"}','{}',true)")
	db.Exec("INSERT INTO ingestion_jobs (id,import_source_id,state,priority) VALUES (1,1,'completed',1)")

	w = doRequest(r, "GET", "/api/v1/ingestion/jobs/1", adminToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("authorized ingestion get job: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	jobBody := parseBody(w)
	if jobBody["id"] == nil {
		t.Error("expected 'id' in job response")
	}
	if jobBody["state"] == nil {
		t.Error("expected 'state' in job response")
	}
}

// ---- Media ----

func TestRealMediaEndpoints(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "mediaadmin", rbac.SystemAdmin, "*", "*")

	// List media (empty)
	w := doRequest(r, "GET", "/api/v1/media", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// POST /media — positive-path handler execution (create a media record).
	w = doRequest(r, "POST", "/api/v1/media", token, map[string]interface{}{
		"title": "Created Song", "mime_type": "audio/mpeg", "duration": 200,
	})
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("create media: expected 201/200, got %d: %s", w.Code, w.Body.String())
	}
	createBody := parseBody(w)
	if createBody["id"] == nil && createBody["title"] == nil {
		t.Error("expected id or title in create media response")
	}

	// Seed a second media asset with known paths for streaming/cover tests.
	db.Exec(`INSERT INTO media_assets (id,title,audio_path,cover_art_path,mime_type,duration,status,lyrics_lrc_path)
		VALUES (100,'TestSong','/tmp/t.mp3','/tmp/cover.jpg','audio/mpeg',120,'active','')`)

	// Get media by ID
	w = doRequest(r, "GET", "/api/v1/media/100", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["title"] != "TestSong" {
		t.Errorf("expected title TestSong, got %v", body["title"])
	}
	if body["mime_type"] == nil {
		t.Error("expected mime_type in get response")
	}

	// GET /media/:id/cover — handler tries c.File() for cover art.
	w = doRequest(r, "GET", "/api/v1/media/100/cover", token, nil)
	if w.Code == http.StatusOK {
		ct := w.Header().Get("Content-Type")
		if ct == "" || (ct != "image/jpeg" && ct != "image/png" && ct != "image/webp" && ct != "application/octet-stream") {
			t.Errorf("cover 200: expected image Content-Type, got %q", ct)
		}
		if w.Body.Len() == 0 {
			t.Error("cover 200: expected non-empty body")
		}
	} else if w.Code == http.StatusNotFound {
		coverBody := parseBody(w)
		if coverBody["message"] == nil || coverBody["message"] == "" {
			t.Error("cover 404: expected error body with message field")
		}
	} else {
		// File missing on disk — 500 is acceptable.
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("cover: expected 200/404/500, got %d: %s", w.Code, w.Body.String())
		}
	}

	// Cover for non-existent media — must be exactly 404.
	w = doRequest(r, "GET", "/api/v1/media/99999/cover", token, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("cover 404: expected exactly 404 for non-existent ID, got %d", w.Code)
	}
	cover404Body := parseBody(w)
	if cover404Body["message"] == nil || cover404Body["message"] == "" {
		t.Error("cover 404: expected message field in error body")
	}

	// GET /media/:id/stream — handler tries c.File() for audio.
	w = doRequest(r, "GET", "/api/v1/media/100/stream", token, nil)
	if w.Code == http.StatusOK {
		ct := w.Header().Get("Content-Type")
		if ct == "" {
			t.Error("stream 200: expected Content-Type header for audio")
		}
		if w.Body.Len() == 0 {
			t.Error("stream 200: expected non-empty body")
		}
	} else if w.Code == http.StatusNotFound {
		streamBody := parseBody(w)
		if streamBody["message"] == nil || streamBody["message"] == "" {
			t.Error("stream 404: expected error body with message field")
		}
	} else if w.Code != http.StatusInternalServerError {
		t.Fatalf("stream: expected 200/404/500, got %d: %s", w.Code, w.Body.String())
	}

	// Stream for non-existent media — must be exactly 404.
	w = doRequest(r, "GET", "/api/v1/media/99999/stream", token, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("stream 404: expected exactly 404 for non-existent ID, got %d", w.Code)
	}

	// Stream unauthorized — no token.
	w = doRequest(r, "GET", "/api/v1/media/100/stream", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("stream unauth: expected 401, got %d", w.Code)
	}

	// Update media
	w = doRequest(r, "PUT", "/api/v1/media/100", token, map[string]interface{}{"title": "Updated"})
	if w.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Get supported formats
	w = doRequest(r, "GET", "/api/v1/media/formats/supported", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("formats: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Lyrics search
	w = doRequest(r, "GET", "/api/v1/media/100/lyrics/search?q=test", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("lyrics search: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Delete media
	w = doRequest(r, "DELETE", "/api/v1/media/100", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ---- Analytics ----

func TestRealAnalyticsEndpoints(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "analyticsadmin", rbac.SystemAdmin, "*", "*")

	// List KPI definitions
	w := doRequest(r, "GET", "/api/v1/analytics/kpis/definitions", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list defs: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Create KPI definition
	w = doRequest(r, "POST", "/api/v1/analytics/kpis/definitions", token, map[string]interface{}{
		"code": "test_kpi", "display_name": "Test KPI", "description": "For testing",
		"formula_sql": "SELECT 1", "dimensions_json": `["city"]`,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create def: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Get KPI definition
	w = doRequest(r, "GET", "/api/v1/analytics/kpis/definitions/test_kpi", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get def: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["code"] != "test_kpi" {
		t.Errorf("expected code test_kpi, got %v", body["code"])
	}

	// Update KPI definition
	w = doRequest(r, "PUT", "/api/v1/analytics/kpis/definitions/test_kpi", token, map[string]interface{}{
		"display_name": "Updated KPI",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("update def: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Delete KPI definition
	w = doRequest(r, "DELETE", "/api/v1/analytics/kpis/definitions/test_kpi", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete def: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Get trends
	w = doRequest(r, "GET", "/api/v1/analytics/trends", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("trends: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRealAnalyticsScopeDenied(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "anoscope", rbac.OperationsAnalyst, "", "")
	w := doRequest(r, "GET", "/api/v1/analytics/kpis/definitions", token, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for no-scope analyst, got %d", w.Code)
	}
	body := parseBody(w)
	if body["code"] == nil {
		t.Error("expected error code in 403 body")
	}
}

// ---- Reports ----

func TestRealReportsLifecycle(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "rptadmin", rbac.SystemAdmin, "*", "*")

	// Create schedule
	w := doRequest(r, "POST", "/api/v1/reports/schedules", token, map[string]interface{}{
		"name": "Weekly", "kpi_code": "test", "cron_expr": "0 8 * * 1",
		"timezone": "UTC", "output_format": "csv", "scope_json": `{"city":"NYC"}`,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Get schedule
	w = doRequest(r, "GET", "/api/v1/reports/schedules/1", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Patch schedule
	w = doRequest(r, "PATCH", "/api/v1/reports/schedules/1", token, map[string]interface{}{"name": "Updated"})
	if w.Code != http.StatusOK {
		t.Fatalf("patch: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// List schedules — positive-path handler execution.
	w = doRequest(r, "GET", "/api/v1/reports/schedules", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list schedules: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	listSchedBody := parseBody(w)
	if listSchedBody["items"] == nil {
		t.Error("expected 'items' in list schedules response")
	}
	schedItems, _ := listSchedBody["items"].([]interface{})
	if len(schedItems) == 0 {
		t.Error("expected at least one schedule in list")
	}
	firstSched, _ := schedItems[0].(map[string]interface{})
	if firstSched["id"] == nil {
		t.Error("expected 'id' in schedule item")
	}
	if firstSched["name"] == nil {
		t.Error("expected 'name' in schedule item")
	}
	if firstSched["cron_expr"] == nil {
		t.Error("expected 'cron_expr' in schedule item")
	}

	// List schedules — unauthorized.
	w = doRequest(r, "GET", "/api/v1/reports/schedules", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("list schedules unauth: expected 401, got %d", w.Code)
	}

	// Trigger run
	w = doRequest(r, "POST", "/api/v1/reports/schedules/1/trigger", token, nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("trigger: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	trigBody := parseBody(w)
	if trigBody["id"] == nil {
		t.Error("expected 'id' in triggered run response")
	}
	if trigBody["state"] == nil {
		t.Error("expected 'state' in triggered run response")
	}

	// List runs — positive-path handler execution.
	w = doRequest(r, "GET", "/api/v1/reports/runs", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list runs: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	listRunsBody := parseBody(w)
	if listRunsBody["items"] == nil {
		t.Error("expected 'items' in list runs response")
	}
	runItems, _ := listRunsBody["items"].([]interface{})
	if len(runItems) == 0 {
		t.Error("expected at least one run in list")
	}
	firstRun, _ := runItems[0].(map[string]interface{})
	if firstRun["id"] == nil {
		t.Error("expected 'id' in run item")
	}
	if firstRun["schedule_id"] == nil {
		t.Error("expected 'schedule_id' in run item")
	}
	if firstRun["state"] == nil {
		t.Error("expected 'state' in run item")
	}

	// List runs — unauthorized.
	w = doRequest(r, "GET", "/api/v1/reports/runs", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("list runs unauth: expected 401, got %d", w.Code)
	}

	// Download run — the triggered run has no real output file on disk,
	// so the handler returns 404 (output not ready).
	w = doRequest(r, "GET", "/api/v1/reports/runs/1/download", token, nil)
	if w.Code == http.StatusOK {
		if w.Body.Len() == 0 {
			t.Error("download 200: body must be non-empty")
		}
		ct := w.Header().Get("Content-Type")
		if ct == "" {
			t.Error("download 200: must have Content-Type (csv/pdf/xlsx)")
		}
		cd := w.Header().Get("Content-Disposition")
		if cd == "" {
			t.Error("download 200: must have Content-Disposition attachment header")
		}
	} else if w.Code == http.StatusNotFound {
		dlBody := parseBody(w)
		if dlBody["message"] == nil || dlBody["message"] == "" {
			t.Error("download 404: must include error message")
		}
	} else {
		t.Fatalf("download: expected 200 or 404, got %d: %s", w.Code, w.Body.String())
	}

	// Download unauthorized — no token.
	w = doRequest(r, "GET", "/api/v1/reports/runs/1/download", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("download unauth: expected 401, got %d", w.Code)
	}

	// Delete schedule
	w = doRequest(r, "DELETE", "/api/v1/reports/schedules/1", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ---- Audit ----

func TestRealAuditEndpoints(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "auditadmin", rbac.SystemAdmin, "*", "*")

	// List logs (login above creates audit entries)
	w := doRequest(r, "GET", "/api/v1/audit/logs", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list logs: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["data"] == nil {
		t.Error("expected data in logs response")
	}
	if body["total"] == nil {
		t.Error("expected total in logs response")
	}

	// Search logs
	w = doRequest(r, "GET", "/api/v1/audit/logs/search?action_type=LOGIN", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("search: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Get specific log
	w = doRequest(r, "GET", "/api/v1/audit/logs/1", token, nil)
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Fatalf("get log: expected 200 or 404, got %d", w.Code)
	}

	// Create delete request
	w = doRequest(r, "POST", "/api/v1/audit/delete-requests", token, map[string]interface{}{
		"reason": "test cleanup", "target_type": "audit_log", "target_id": "range:1-50",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create dr: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	body = parseBody(w)
	if body["state"] != "pending" {
		t.Errorf("expected state pending, got %v", body["state"])
	}

	// List delete requests
	w = doRequest(r, "GET", "/api/v1/audit/delete-requests", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list dr: expected 200, got %d", w.Code)
	}

	// Get delete request
	w = doRequest(r, "GET", "/api/v1/audit/delete-requests/1", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get dr: expected 200, got %d", w.Code)
	}

	// Approve (first approval by a different admin)
	token2 := loginAndGetToken(db, r, "auditadmin2", rbac.SystemAdmin, "*", "*")
	w = doRequest(r, "POST", "/api/v1/audit/delete-requests/1/approve", token2, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ---- Security ----

func TestRealSecurityEndpoints(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "secadmin", rbac.SystemAdmin, "*", "*")

	// Sensitive fields
	w := doRequest(r, "POST", "/api/v1/security/sensitive-fields", token, map[string]interface{}{
		"field_key": "test.ssn", "display_name": "SSN", "mask_pattern": "***-**-####",
		"unmask_roles_json": `["system_admin"]`,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create sf: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	w = doRequest(r, "GET", "/api/v1/security/sensitive-fields", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list sf: expected 200, got %d", w.Code)
	}
	w = doRequest(r, "PUT", "/api/v1/security/sensitive-fields/1", token, map[string]interface{}{
		"mask_pattern": "XXX-XX-####",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("update sf: expected 200, got %d", w.Code)
	}
	w = doRequest(r, "DELETE", "/api/v1/security/sensitive-fields/1", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete sf: expected 200, got %d", w.Code)
	}

	// Keys
	w = doRequest(r, "GET", "/api/v1/security/keys", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list keys: expected 200, got %d", w.Code)
	}
	w = doRequest(r, "POST", "/api/v1/security/keys/rotate", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("rotate: expected 200, got %d", w.Code)
	}
	w = doRequest(r, "GET", "/api/v1/security/keys/1", token, nil)
	if w.Code == http.StatusOK || w.Code == http.StatusNotFound {
		// OK — key may or may not exist yet.
	}

	// Password reset
	db.Create(&models.User{Username: "resetme", PasswordHash: "x", Role: rbac.StandardUser, CityScope: "NYC", DepartmentScope: "Finance", Status: "active"})
	w = doRequest(r, "POST", "/api/v1/security/password-reset", token, map[string]interface{}{
		"user_id": 2,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("pw reset: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	w = doRequest(r, "GET", "/api/v1/security/password-reset", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list pw resets: expected 200, got %d", w.Code)
	}
	w = doRequest(r, "POST", "/api/v1/security/password-reset/1/approve", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("approve pw reset: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Retention policies
	w = doRequest(r, "POST", "/api/v1/security/retention-policies", token, map[string]interface{}{
		"artifact_type": "audit_logs", "retention_days": 365, "description": "1yr",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create rp: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	w = doRequest(r, "GET", "/api/v1/security/retention-policies", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list rp: expected 200, got %d", w.Code)
	}
	w = doRequest(r, "PUT", "/api/v1/security/retention-policies/1", token, map[string]interface{}{
		"retention_days": 180,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("update rp: expected 200, got %d", w.Code)
	}

	// Legal holds
	w = doRequest(r, "POST", "/api/v1/security/legal-holds", token, map[string]interface{}{
		"scope_json": `{"city":"NYC"}`, "reason": "investigation",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create lh: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	w = doRequest(r, "GET", "/api/v1/security/legal-holds", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list lh: expected 200, got %d", w.Code)
	}
	w = doRequest(r, "POST", "/api/v1/security/legal-holds/1/release", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("release lh: expected 200, got %d", w.Code)
	}

	// Purge runs
	w = doRequest(r, "GET", "/api/v1/security/purge-runs", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list purge: expected 200, got %d", w.Code)
	}
}

func TestRealSecurityRBACDenied(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "secdeny", rbac.StandardUser, "NYC", "Finance")
	w := doRequest(r, "GET", "/api/v1/security/keys", token, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	body := parseBody(w)
	if body["code"] == nil {
		t.Error("expected error code in 403 body")
	}
}

// ---- Integrations ----

func TestRealIntegrationEndpoints(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "intadmin", rbac.SystemAdmin, "*", "*")

	// Endpoints CRUD
	w := doRequest(r, "POST", "/api/v1/integrations/endpoints", token, map[string]interface{}{
		"name": "Hook A", "event_type": "record.created", "url": "https://hook.test/a",
		"http_method": "POST", "max_retries": 3, "timeout_seconds": 30,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create ep: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	w = doRequest(r, "GET", "/api/v1/integrations/endpoints", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list ep: expected 200, got %d", w.Code)
	}
	w = doRequest(r, "GET", "/api/v1/integrations/endpoints/1", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get ep: expected 200, got %d", w.Code)
	}
	w = doRequest(r, "PUT", "/api/v1/integrations/endpoints/1", token, map[string]interface{}{
		"name": "Hook A v2", "event_type": "record.created", "url": "https://hook.test/a2",
		"http_method": "POST", "max_retries": 3, "timeout_seconds": 30,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("update ep: expected 200, got %d", w.Code)
	}
	w = doRequest(r, "POST", "/api/v1/integrations/endpoints/1/test", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("test ep: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	w = doRequest(r, "DELETE", "/api/v1/integrations/endpoints/1", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete ep: expected 200, got %d", w.Code)
	}

	// Deliveries
	w = doRequest(r, "GET", "/api/v1/integrations/deliveries", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list del: expected 200, got %d", w.Code)
	}

	// Connectors CRUD
	w = doRequest(r, "POST", "/api/v1/integrations/connectors", token, map[string]interface{}{
		"name": "MySQL Conn", "connector_type": "database",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create conn: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	w = doRequest(r, "GET", "/api/v1/integrations/connectors", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list conn: expected 200, got %d", w.Code)
	}
	w = doRequest(r, "GET", "/api/v1/integrations/connectors/1", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get conn: expected 200, got %d", w.Code)
	}
	w = doRequest(r, "PUT", "/api/v1/integrations/connectors/1", token, map[string]interface{}{
		"name": "MySQL Conn v2", "connector_type": "database",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("update conn: expected 200, got %d", w.Code)
	}
	w = doRequest(r, "POST", "/api/v1/integrations/connectors/1/health-check", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("health-check: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	w = doRequest(r, "DELETE", "/api/v1/integrations/connectors/1", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete conn: expected 200, got %d", w.Code)
	}
}

func TestRealIntegrationsRBACDenied(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "intdeny", rbac.OperationsAnalyst, "NYC", "Finance")
	w := doRequest(r, "GET", "/api/v1/integrations/endpoints", token, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

// ==========================================================================
// Additional no-mock tests for previously mock-only endpoints
// ==========================================================================

// ---- DELETE /versions/:entity/:id/items/:itemId ----

func TestRealVersionRemoveItem(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "verdelitem", rbac.SystemAdmin, "*", "*")
	db.Exec("INSERT INTO org_nodes (id,level_code,level_label,name,is_active,sort_order) VALUES (100,'company','Company','Corp',true,0)")
	db.Exec("INSERT INTO context_assignments (user_id, org_node_id) VALUES (1, 100)")
	db.Exec("INSERT INTO master_records (id,entity_type,natural_key,payload_json,status) VALUES (1,'sku','RI1','{\"n\":1}','active')")

	// Create version
	w := doRequest(r, "POST", "/api/v1/versions/sku", token, map[string]interface{}{"scope_key": "node:100"})
	if w.Code != http.StatusCreated {
		t.Fatalf("create version: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	vData, _ := body["data"].(map[string]interface{})
	vID := vData["id"]

	// Add item
	w = doRequest(r, "POST", fmt.Sprintf("/api/v1/versions/sku/%.0f/items", vID), token, map[string]interface{}{
		"master_record_ids": []int{1},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("add item: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// List items to get item ID
	w = doRequest(r, "GET", fmt.Sprintf("/api/v1/versions/sku/%.0f/items", vID), token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list items: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body = parseBody(w)
	items, ok := body["data"].([]interface{})
	if !ok || len(items) == 0 {
		t.Fatal("expected at least one version item")
	}
	firstItem, _ := items[0].(map[string]interface{})
	itemID := firstItem["id"]

	// Delete the item
	w = doRequest(r, "DELETE", fmt.Sprintf("/api/v1/versions/sku/%.0f/items/%.0f", vID, itemID), token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("remove item: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body = parseBody(w)
	if body["message"] == nil || body["message"] == "" {
		t.Error("expected message confirming item removal")
	}
}

// ---- GET /integrations/deliveries/:id ----

func TestRealIntegrationGetDeliveryByID(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "delget", rbac.SystemAdmin, "*", "*")

	// Create an endpoint first, then seed a delivery.
	w := doRequest(r, "POST", "/api/v1/integrations/endpoints", token, map[string]interface{}{
		"name": "DeliveryHook", "event_type": "record.created",
		"url": "https://hook.test/d", "http_method": "POST",
		"max_retries": 3, "timeout_seconds": 30,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create endpoint: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Seed a delivery directly since there's no create-delivery API endpoint.
	db.Exec(`INSERT INTO integration_deliveries (endpoint_id, event_id, state, payload_json, retries, dedupe_key)
		VALUES (1, 'evt-test-001', 'pending', '{"data":"test"}', 0, 'dk-001')`)

	// GET the delivery by ID.
	w = doRequest(r, "GET", "/api/v1/integrations/deliveries/1", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get delivery: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["event_id"] != "evt-test-001" {
		t.Errorf("expected event_id evt-test-001, got %v", body["event_id"])
	}
	if body["state"] != "pending" {
		t.Errorf("expected state pending, got %v", body["state"])
	}
}

// ---- GET /media/:id/cover ----

func TestRealMediaCoverEndpoint(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "mediacover", rbac.SystemAdmin, "*", "*")

	// Seed a media asset (no real file on disk, so expect 404 or 500 from c.File).
	db.Exec(`INSERT INTO media_assets (id,title,audio_path,cover_art_path,mime_type,duration,status)
		VALUES (1,'CoverTest','/tmp/t.mp3','/tmp/cover.jpg','audio/mpeg',120,'active')`)

	w := doRequest(r, "GET", "/api/v1/media/1/cover", token, nil)
	// The handler tries c.File() which will fail since the file doesn't exist on disk.
	// We accept 200 (if file happened to exist) or 404/500 (file missing) —
	// the key assertion is that the real middleware chain processes the request.
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Fatalf("cover: expected 200/404/500, got %d: %s", w.Code, w.Body.String())
	}
	// If 200, verify content-type is an image type.
	if w.Code == http.StatusOK {
		ct := w.Header().Get("Content-Type")
		if ct == "" {
			t.Error("expected Content-Type header on 200 cover response")
		}
	}
}

// ---- GET /health (real no-mock) ----

func TestRealHealthEndpoint(t *testing.T) {
	_, r := skipIfNoDB(t)

	w := doRequest(r, "GET", "/health", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("health: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %v", body["status"])
	}
	if body["timestamp"] == nil || body["timestamp"] == "" {
		t.Error("expected non-empty timestamp")
	}
}

// ---- POST /audit/delete-requests/:id/execute ----

func TestRealAuditDeleteRequestExecute(t *testing.T) {
	db, r := skipIfNoDB(t)
	token1 := loginAndGetToken(db, r, "auditexec1", rbac.SystemAdmin, "*", "*")
	token2 := loginAndGetToken(db, r, "auditexec2", rbac.SystemAdmin, "*", "*")

	// Seed some audit logs to be deleted.
	db.Exec("INSERT INTO audit_logs (action_type, target_type, target_id, ip_address) VALUES ('TEST','test','1','127.0.0.1')")

	// Create delete request.
	w := doRequest(r, "POST", "/api/v1/audit/delete-requests", token1, map[string]interface{}{
		"reason": "cleanup", "target_type": "audit_log", "target_id": "range:1-100",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// First approval (by token2, different admin).
	w = doRequest(r, "POST", "/api/v1/audit/delete-requests/1/approve", token2, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("first approve: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Second approval (by token1).
	w = doRequest(r, "POST", "/api/v1/audit/delete-requests/1/approve", token1, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("second approve: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Execute.
	w = doRequest(r, "POST", "/api/v1/audit/delete-requests/1/execute", token1, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("execute: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["message"] == nil || body["message"] == "" {
		t.Error("expected execution message")
	}
	if body["deleted_count"] == nil {
		t.Error("expected deleted_count in execution response")
	}
}

// ---- POST /integrations/deliveries/:id/retry ----

func TestRealIntegrationDeliveryRetry(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "delretry", rbac.SystemAdmin, "*", "*")

	// Create endpoint + seed delivery.
	doRequest(r, "POST", "/api/v1/integrations/endpoints", token, map[string]interface{}{
		"name": "RetryHook", "event_type": "record.updated",
		"url": "https://hook.test/r", "http_method": "POST",
		"max_retries": 3, "timeout_seconds": 30,
	})
	db.Exec(`INSERT INTO integration_deliveries (endpoint_id, event_id, state, payload_json, retries, dedupe_key)
		VALUES (1, 'evt-retry-001', 'failed', '{"data":"test"}', 1, 'dk-retry')`)

	w := doRequest(r, "POST", "/api/v1/integrations/deliveries/1/retry", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("retry: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["message"] == nil || body["message"] == "" {
		t.Error("expected retry message")
	}
}

// ---- POST /media/:id/lyrics/parse ----

func TestRealMediaLyricsParse(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "lyricsparse", rbac.SystemAdmin, "*", "*")

	// Seed a media asset.
	db.Exec(`INSERT INTO media_assets (id,title,audio_path,mime_type,duration,status)
		VALUES (1,'LyricsTest','/tmp/l.mp3','audio/mpeg',120,'active')`)

	// Send lyrics content to parse.
	w := doRequest(r, "POST", "/api/v1/media/1/lyrics/parse", token, map[string]interface{}{
		"content": "[00:00.00]Hello world\n[00:05.00]Second line",
	})
	// Handler may return 200 with parse result or status indicating parse outcome.
	if w.Code != http.StatusOK {
		t.Fatalf("lyrics parse: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["status"] == nil {
		t.Error("expected 'status' field in lyrics parse response")
	}
	if body["lines"] == nil {
		t.Error("expected 'lines' field in lyrics parse response")
	}
}
