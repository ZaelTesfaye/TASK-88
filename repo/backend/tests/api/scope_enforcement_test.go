//go:build integration

package api

import (
	"net/http"
	"testing"

	"backend/internal/rbac"
)

// All tests use realRouter(db) with loginAndGetToken — no fakeAuthMiddleware.

func TestGetMasterRecordCrossScopeDenied(t *testing.T) {
	db, r := skipIfNoDB(t)

	tokenA := loginAndGetToken(db, r, "analyst_a", rbac.OperationsAnalyst, "CityA", "Finance")
	tokenB := loginAndGetToken(db, r, "analyst_b", rbac.OperationsAnalyst, "CityB", "Engineering")

	db.Exec("INSERT INTO org_nodes (id, level_code, level_label, name, city, department) VALUES (100, 'city', 'City', 'City A', 'CityA', 'Finance')")
	db.Exec("INSERT INTO org_nodes (id, level_code, level_label, name, city, department) VALUES (101, 'city', 'City', 'City B', 'CityB', 'Engineering')")
	// Assign userA (id=1) to node 100, userB (id=2) to node 101.
	db.Exec("INSERT INTO context_assignments (user_id, org_node_id) VALUES (1, 100)")
	db.Exec("INSERT INTO context_assignments (user_id, org_node_id) VALUES (2, 101)")

	db.Exec("INSERT INTO master_records (id, entity_type, natural_key, status) VALUES (1, 'sku', 'TEST001', 'active')")
	db.Exec("INSERT INTO master_versions (id, entity_type, scope_key, version_no, state, created_by) VALUES (1, 'sku', 'node:101', 1, 'active', 2)")
	db.Exec("INSERT INTO master_version_items (version_id, master_record_id) VALUES (1, 1)")

	// User A (CityA) cannot access record scoped to CityB.
	w := doRequest(r, "GET", "/api/v1/master/sku/1", tokenA, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cross-scope GET, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["code"] != "FORBIDDEN" {
		t.Errorf("expected error code 'FORBIDDEN', got %v", body["code"])
	}
	msg, _ := body["message"].(string)
	if msg == "" {
		t.Error("expected non-empty error message in 403 response")
	}

	// User B (CityB) CAN access the record.
	w2 := doRequest(r, "GET", "/api/v1/master/sku/1", tokenB, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for in-scope GET, got %d: %s", w2.Code, w2.Body.String())
	}
	body2 := parseBody(w2)
	if body2["data"] == nil {
		t.Error("expected data in 200 response")
	}
}

func TestUpdateMasterRecordCrossScopeDenied(t *testing.T) {
	db, r := skipIfNoDB(t)

	tokenA := loginAndGetToken(db, r, "steward_a", rbac.DataSteward, "CityA", "Finance")

	db.Exec("INSERT INTO org_nodes (id, level_code, level_label, name, city, department) VALUES (100, 'city', 'City', 'City A', 'CityA', 'Finance')")
	db.Exec("INSERT INTO org_nodes (id, level_code, level_label, name, city, department) VALUES (101, 'city', 'City', 'City B', 'CityB', 'HR')")
	db.Exec("INSERT INTO context_assignments (user_id, org_node_id) VALUES (1, 100)")

	db.Exec("INSERT INTO master_records (id, entity_type, natural_key, status) VALUES (1, 'sku', 'TESTUPD001', 'active')")
	db.Exec("INSERT INTO master_versions (id, entity_type, scope_key, version_no, state, created_by) VALUES (1, 'sku', 'node:101', 1, 'active', 1)")
	db.Exec("INSERT INTO master_version_items (version_id, master_record_id) VALUES (1, 1)")

	w := doRequest(r, "PUT", "/api/v1/master/sku/1", tokenA, map[string]interface{}{"payload_json": `{"updated": true}`})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cross-scope PUT, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["code"] != "FORBIDDEN" {
		t.Errorf("expected error code 'FORBIDDEN', got %v", body["code"])
	}
	msg, _ := body["message"].(string)
	if msg == "" || msg == "null" {
		t.Error("expected non-empty scope denial message")
	}
}

func TestNoContextAssignmentDenied(t *testing.T) {
	db, r := skipIfNoDB(t)

	tokenA := loginAndGetToken(db, r, "analyst_no_ctx", rbac.OperationsAnalyst, "CityA", "Finance")

	db.Exec("INSERT INTO org_nodes (id, level_code, level_label, name, city, department) VALUES (100, 'city', 'City', 'City A', 'CityA', 'Finance')")
	// No context_assignments row.

	db.Exec("INSERT INTO master_records (id, entity_type, natural_key, status) VALUES (1, 'sku', 'TESTNOCTX', 'active')")
	db.Exec("INSERT INTO master_versions (id, entity_type, scope_key, version_no, state, created_by) VALUES (1, 'sku', 'node:100', 1, 'active', 1)")
	db.Exec("INSERT INTO master_version_items (version_id, master_record_id) VALUES (1, 1)")

	w := doRequest(r, "GET", "/api/v1/master/sku/1", tokenA, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for user with no context assignment, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAnalyticsKPIScopedResults(t *testing.T) {
	db, r := skipIfNoDB(t)

	token := loginAndGetToken(db, r, "ops_nyc_kpi", rbac.OperationsAnalyst, "NYC", "Finance")

	db.Exec("INSERT INTO org_nodes (id, level_code, level_label, name, city, department) VALUES (100, 'city', 'City', 'NYC', 'NYC', 'Finance')")
	db.Exec("INSERT INTO context_assignments (user_id, org_node_id) VALUES (1, 100)")

	w := doRequest(r, "GET", "/api/v1/analytics/kpis", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["kpis"] == nil {
		t.Error("expected 'kpis' field in response body")
	}
	kpis, ok := body["kpis"].([]interface{})
	if !ok {
		t.Fatal("expected 'kpis' to be an array")
	}
	_ = kpis
}

func TestReportGetRunCrossScopeDenied(t *testing.T) {
	db, r := skipIfNoDB(t)

	token := loginAndGetToken(db, r, "ops_nyc_rpt", rbac.OperationsAnalyst, "NYC", "Finance")

	db.Exec("INSERT INTO org_nodes (id, level_code, level_label, name, city, department) VALUES (100, 'city', 'City', 'NYC', 'NYC', 'Finance')")
	db.Exec("INSERT INTO context_assignments (user_id, org_node_id) VALUES (1, 100)")

	db.Exec("INSERT INTO report_schedules (id, name, cron_expr, timezone, output_format, scope_json, created_by) VALUES (1, 'LAX Report', '0 8 * * 1', 'UTC', 'csv', '{\"city\":\"LAX\"}', 1)")
	db.Exec("INSERT INTO report_runs (id, schedule_id, state, row_count) VALUES (1, 1, 'ready', 10)")

	w := doRequest(r, "GET", "/api/v1/reports/runs/1", token, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cross-scope run, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["message"] == nil {
		t.Error("expected error message in cross-scope 403")
	}
}

func TestReportAccessCheckCrossScopeDenied(t *testing.T) {
	db, r := skipIfNoDB(t)

	token := loginAndGetToken(db, r, "ops_chi_ac", rbac.OperationsAnalyst, "CHI", "Sales")

	db.Exec("INSERT INTO org_nodes (id, level_code, level_label, name, city, department) VALUES (100, 'city', 'City', 'CHI', 'CHI', 'Sales')")
	db.Exec("INSERT INTO context_assignments (user_id, org_node_id) VALUES (1, 100)")

	db.Exec("INSERT INTO report_schedules (id, name, cron_expr, timezone, output_format, scope_json, created_by) VALUES (1, 'NYC Report', '0 8 * * 1', 'UTC', 'csv', '{\"city\":\"NYC\"}', 1)")
	db.Exec("INSERT INTO report_runs (id, schedule_id, state, row_count) VALUES (1, 1, 'ready', 5)")

	w := doRequest(r, "GET", "/api/v1/reports/runs/1/access-check", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from access-check, got %d", w.Code)
	}
	resp := parseBody(w)
	hasAccess, ok := resp["has_access"].(bool)
	if !ok || hasAccess {
		t.Fatalf("expected has_access=false, got %v", resp["has_access"])
	}
}

func TestMediaMutationsRequireElevatedRole(t *testing.T) {
	db, r := skipIfNoDB(t)

	stdToken := loginAndGetToken(db, r, "media_std", rbac.StandardUser, "NYC", "Finance")
	stewardToken := loginAndGetToken(db, r, "media_steward", rbac.DataSteward, "NYC", "Finance")

	// standard_user GET /media → 200
	w := doRequest(r, "GET", "/api/v1/media", stdToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("standard_user GET /media: expected 200, got %d", w.Code)
	}

	// standard_user POST /media → 403
	w = doRequest(r, "POST", "/api/v1/media", stdToken, map[string]interface{}{"title": "test"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("standard_user POST /media: expected 403, got %d", w.Code)
	}
	body := parseBody(w)
	if body["message"] == nil {
		t.Error("expected error message in 403 response")
	}

	// standard_user DELETE /media/1 → 403
	w = doRequest(r, "DELETE", "/api/v1/media/1", stdToken, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("standard_user DELETE /media: expected 403, got %d", w.Code)
	}

	// data_steward can list media → 200
	w = doRequest(r, "GET", "/api/v1/media", stewardToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("data_steward GET /media: expected 200, got %d", w.Code)
	}
}
