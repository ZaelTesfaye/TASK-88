//go:build integration

package api

import (
	"net/http"
	"testing"

	"backend/internal/rbac"
)

// All security-regression tests use realRouter(db) + loginAndGetToken.
// Zero fakeAuthMiddleware or signToken usage.

func TestAuditLogPurgeViaSecurityEndpointRejected(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "admin_purge", rbac.SystemAdmin, "*", "*")

	w := doRequest(r, "POST", "/api/v1/security/purge-runs/execute", token, map[string]interface{}{
		"artifact_type": "audit_logs",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for audit_logs purge, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(w)
	if resp["message"] == nil || resp["message"] == "" {
		t.Error("expected message explaining rejection")
	}
}

func TestAuditLogDryRunViaSecurityEndpointRejected(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "admin_dryrun", rbac.SystemAdmin, "*", "*")

	w := doRequest(r, "POST", "/api/v1/security/purge-runs/dry-run", token, map[string]interface{}{
		"artifact_type": "audit_logs",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for audit_logs dry-run, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(w)
	if resp["message"] == nil || resp["message"] == "" {
		t.Error("expected error message in dry-run rejection")
	}
	if resp["code"] == nil {
		t.Error("expected error code in dry-run rejection")
	}
}

func TestNonAuditLogPurgeAllowed(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "admin_ok_purge", rbac.SystemAdmin, "*", "*")

	db.Exec("INSERT INTO retention_policies (artifact_type, retention_days, is_active) VALUES ('ingestion_failures', 90, true)")

	w := doRequest(r, "POST", "/api/v1/security/purge-runs/execute", token, map[string]interface{}{
		"artifact_type": "ingestion_failures",
	})
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("expected 200/201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAnalyticsScopeFromAuthContext(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "ops_nyc_sc", rbac.OperationsAnalyst, "NYC", "Finance")

	db.Exec("INSERT INTO org_nodes (id, level_code, level_label, name, city, department) VALUES (100, 'city', 'City', 'NYC', 'NYC', 'Finance')")
	db.Exec("INSERT INTO context_assignments (user_id, org_node_id) VALUES (1, 100)")

	w := doRequest(r, "GET", "/api/v1/analytics/kpis?city_scope=LAX&dept_scope=HR", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["kpis"] == nil {
		t.Error("expected 'kpis' field")
	}
	if _, ok := body["kpis"].([]interface{}); !ok {
		t.Error("expected 'kpis' to be an array")
	}
}

func TestAnalyticsNoScopeUserDenied(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "ops_noscope_sr", rbac.OperationsAnalyst, "", "")

	w := doRequest(r, "GET", "/api/v1/analytics/kpis", token, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["message"] == nil || body["message"] == "" {
		t.Error("expected error message")
	}
	if body["code"] == nil {
		t.Error("expected error code")
	}
}

func TestAnalyticsSystemAdminBypassesScope(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "admin_scope_bp", rbac.SystemAdmin, "", "")

	w := doRequest(r, "GET", "/api/v1/analytics/kpis", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["kpis"] == nil {
		t.Error("expected 'kpis' field")
	}
}

func TestNoScopeUserDeniedAnalyticsAndReports(t *testing.T) {
	db, r := skipIfNoDB(t)
	token := loginAndGetToken(db, r, "ops_empty_sr", rbac.OperationsAnalyst, "", "")

	w := doRequest(r, "GET", "/api/v1/analytics/kpis", token, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for analytics, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	if body["message"] == nil {
		t.Error("expected error message for analytics denial")
	}

	w2 := doRequest(r, "GET", "/api/v1/reports/schedules", token, nil)
	if w2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for reports, got %d: %s", w2.Code, w2.Body.String())
	}
	body2 := parseBody(w2)
	if body2["message"] == nil {
		t.Error("expected error message for reports denial")
	}
}

func TestScopeViolationOnScopedResource(t *testing.T) {
	db, r := skipIfNoDB(t)

	tokenLAX := loginAndGetToken(db, r, "ops_lax_sv", rbac.OperationsAnalyst, "LAX", "Engineering")
	tokenNYC := loginAndGetToken(db, r, "ops_nyc_sv", rbac.OperationsAnalyst, "NYC", "Finance")

	db.Exec("INSERT INTO org_nodes (id, level_code, level_label, name, city, department) VALUES (100, 'city', 'City', 'LAX', 'LAX', 'Engineering')")
	db.Exec("INSERT INTO org_nodes (id, level_code, level_label, name, city, department) VALUES (101, 'city', 'City', 'NYC', 'NYC', 'Finance')")
	db.Exec("INSERT INTO context_assignments (user_id, org_node_id) VALUES (1, 100)")
	db.Exec("INSERT INTO context_assignments (user_id, org_node_id) VALUES (2, 101)")

	db.Exec("INSERT INTO master_records (id, entity_type, natural_key, status) VALUES (1, 'sku', 'SCOPE_TEST', 'active')")
	db.Exec("INSERT INTO master_versions (id, entity_type, scope_key, version_no, state, created_by) VALUES (1, 'sku', 'node:101', 1, 'active', 2)")
	db.Exec("INSERT INTO master_version_items (version_id, master_record_id) VALUES (1, 1)")

	// LAX user → 403
	w := doRequest(r, "GET", "/api/v1/master/sku/1", tokenLAX, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for LAX user on NYC record, got %d", w.Code)
	}

	// NYC user → 200
	w2 := doRequest(r, "GET", "/api/v1/master/sku/1", tokenNYC, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for NYC user, got %d: %s", w2.Code, w2.Body.String())
	}
	body := parseBody(w2)
	if body["data"] == nil {
		t.Error("expected data in 200 response")
	}
}
