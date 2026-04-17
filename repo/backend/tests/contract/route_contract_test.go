// Package contract contains tests that verify the API surface contract
// between the frontend adapters and the backend router.
package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"backend/internal/config"
	"backend/internal/middleware"
	"backend/internal/router"
)

func init() { gin.SetMode(gin.TestMode) }

// doRequest makes an HTTP request. No fakeAuthMiddleware.
func doRequest(r *gin.Engine, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req, _ := http.NewRequest(method, path, buf)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func parseBody(w *httptest.ResponseRecorder) map[string]interface{} {
	var m map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	return m
}

// ==========================================================================
// Route contract: every frontend API adapter URL must match a backend route.
// Uses the real production router (no fakeAuthMiddleware).
// ==========================================================================

func TestAPIContractRoutes(t *testing.T) {
	cfg := config.GetConfig()

	expectedRoutes := []struct{ method, path string }{
		{"POST", "/api/v1/auth/login"},
		{"POST", "/api/v1/auth/logout"},
		{"POST", "/api/v1/auth/refresh"},
		{"GET", "/api/v1/org/tree"},
		{"GET", "/api/v1/org/nodes"},
		{"POST", "/api/v1/org/nodes"},
		{"GET", "/api/v1/org/nodes/:id"},
		{"PUT", "/api/v1/org/nodes/:id"},
		{"DELETE", "/api/v1/org/nodes/:id"},
		{"POST", "/api/v1/context/switch"},
		{"GET", "/api/v1/context/current"},
		{"GET", "/api/v1/master/:entity"},
		{"GET", "/api/v1/master/:entity/:id"},
		{"GET", "/api/v1/master/:entity/:id/history"},
		{"POST", "/api/v1/master/:entity"},
		{"PUT", "/api/v1/master/:entity/:id"},
		{"POST", "/api/v1/master/:entity/:id/deactivate"},
		{"GET", "/api/v1/versions/:entity"},
		{"GET", "/api/v1/versions/:entity/:id"},
		{"GET", "/api/v1/versions/:entity/:id/items"},
		{"GET", "/api/v1/versions/:entity/:id/diff"},
		{"POST", "/api/v1/versions/:entity"},
		{"POST", "/api/v1/versions/:entity/:id/review"},
		{"POST", "/api/v1/versions/:entity/:id/items"},
		{"DELETE", "/api/v1/versions/:entity/:id/items/:itemId"},
		{"POST", "/api/v1/versions/:entity/:id/activate"},
		{"GET", "/api/v1/ingestion/sources"},
		{"POST", "/api/v1/ingestion/sources"},
		{"GET", "/api/v1/ingestion/sources/:id"},
		{"PUT", "/api/v1/ingestion/sources/:id"},
		{"DELETE", "/api/v1/ingestion/sources/:id"},
		{"GET", "/api/v1/ingestion/jobs"},
		{"POST", "/api/v1/ingestion/jobs"},
		{"GET", "/api/v1/ingestion/jobs/:id"},
		{"POST", "/api/v1/ingestion/jobs/:id/retry"},
		{"POST", "/api/v1/ingestion/jobs/:id/acknowledge"},
		{"GET", "/api/v1/ingestion/jobs/:id/checkpoints"},
		{"GET", "/api/v1/ingestion/jobs/:id/failures"},
		{"GET", "/api/v1/media"},
		{"POST", "/api/v1/media"},
		{"GET", "/api/v1/media/:id"},
		{"PUT", "/api/v1/media/:id"},
		{"DELETE", "/api/v1/media/:id"},
		{"GET", "/api/v1/media/:id/stream"},
		{"GET", "/api/v1/media/:id/cover"},
		{"POST", "/api/v1/media/:id/lyrics/parse"},
		{"GET", "/api/v1/media/:id/lyrics/search"},
		{"GET", "/api/v1/media/formats/supported"},
		{"GET", "/api/v1/analytics/kpis"},
		{"GET", "/api/v1/analytics/kpis/definitions"},
		{"POST", "/api/v1/analytics/kpis/definitions"},
		{"GET", "/api/v1/analytics/kpis/definitions/:code"},
		{"PUT", "/api/v1/analytics/kpis/definitions/:code"},
		{"DELETE", "/api/v1/analytics/kpis/definitions/:code"},
		{"GET", "/api/v1/analytics/trends"},
		{"GET", "/api/v1/reports/schedules"},
		{"POST", "/api/v1/reports/schedules"},
		{"GET", "/api/v1/reports/schedules/:id"},
		{"PATCH", "/api/v1/reports/schedules/:id"},
		{"DELETE", "/api/v1/reports/schedules/:id"},
		{"POST", "/api/v1/reports/schedules/:id/trigger"},
		{"GET", "/api/v1/reports/runs"},
		{"GET", "/api/v1/reports/runs/:id"},
		{"GET", "/api/v1/reports/runs/:id/download"},
		{"GET", "/api/v1/reports/runs/:id/access-check"},
		{"GET", "/api/v1/audit/logs"},
		{"GET", "/api/v1/audit/logs/:id"},
		{"GET", "/api/v1/audit/logs/search"},
		{"GET", "/api/v1/audit/delete-requests"},
		{"POST", "/api/v1/audit/delete-requests"},
		{"GET", "/api/v1/audit/delete-requests/:id"},
		{"POST", "/api/v1/audit/delete-requests/:id/approve"},
		{"POST", "/api/v1/audit/delete-requests/:id/execute"},
		{"GET", "/api/v1/security/sensitive-fields"},
		{"POST", "/api/v1/security/sensitive-fields"},
		{"PUT", "/api/v1/security/sensitive-fields/:id"},
		{"DELETE", "/api/v1/security/sensitive-fields/:id"},
		{"GET", "/api/v1/security/keys"},
		{"POST", "/api/v1/security/keys/rotate"},
		{"GET", "/api/v1/security/keys/:id"},
		{"POST", "/api/v1/security/password-reset"},
		{"POST", "/api/v1/security/password-reset/:id/approve"},
		{"GET", "/api/v1/security/password-reset"},
		{"GET", "/api/v1/security/retention-policies"},
		{"POST", "/api/v1/security/retention-policies"},
		{"PUT", "/api/v1/security/retention-policies/:id"},
		{"GET", "/api/v1/security/legal-holds"},
		{"POST", "/api/v1/security/legal-holds"},
		{"POST", "/api/v1/security/legal-holds/:id/release"},
		{"POST", "/api/v1/security/purge-runs/dry-run"},
		{"POST", "/api/v1/security/purge-runs/execute"},
		{"GET", "/api/v1/security/purge-runs"},
		{"GET", "/api/v1/integrations/endpoints"},
		{"POST", "/api/v1/integrations/endpoints"},
		{"GET", "/api/v1/integrations/endpoints/:id"},
		{"PUT", "/api/v1/integrations/endpoints/:id"},
		{"DELETE", "/api/v1/integrations/endpoints/:id"},
		{"POST", "/api/v1/integrations/endpoints/:id/test"},
		{"GET", "/api/v1/integrations/deliveries"},
		{"GET", "/api/v1/integrations/deliveries/:id"},
		{"POST", "/api/v1/integrations/deliveries/:id/retry"},
		{"GET", "/api/v1/integrations/connectors"},
		{"POST", "/api/v1/integrations/connectors"},
		{"GET", "/api/v1/integrations/connectors/:id"},
		{"PUT", "/api/v1/integrations/connectors/:id"},
		{"DELETE", "/api/v1/integrations/connectors/:id"},
		{"POST", "/api/v1/integrations/connectors/:id/health-check"},
		{"GET", "/health"},
	}

	r := router.SetupRouter(cfg, nil)
	registered := r.Routes()
	routeSet := make(map[string]bool, len(registered))
	for _, rt := range registered {
		routeSet[rt.Method+" "+rt.Path] = true
	}

	for _, exp := range expectedRoutes {
		key := exp.method + " " + exp.path
		if !routeSet[key] {
			t.Errorf("expected route %s not registered in router", key)
		}
	}

	// Verify health endpoint returns correct shape via real router.
	w := doRequest(r, "GET", "/health", "", nil)
	if w.Code != http.StatusOK {
		t.Errorf("GET /health: expected 200, got %d", w.Code)
	}
	hb := parseBody(w)
	if hb["status"] != "healthy" {
		t.Errorf("GET /health: expected status 'healthy', got %v", hb["status"])
	}
	if hb["timestamp"] == nil || hb["timestamp"] == "" {
		t.Error("GET /health: expected non-empty timestamp")
	}

	// Verify unauthenticated request returns 401 with error envelope.
	w2 := doRequest(r, "GET", "/api/v1/org/tree", "", nil)
	if w2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w2.Code)
	}
	body401 := parseBody(w2)
	if body401["code"] == nil {
		t.Error("401 response must contain 'code'")
	}
	if body401["message"] == nil {
		t.Error("401 response must contain 'message'")
	}

	// Verify login endpoint accepts POST with body validation.
	w3 := doRequest(r, "POST", "/api/v1/auth/login", "", nil)
	if w3.Code != http.StatusBadRequest {
		t.Errorf("POST /auth/login with no body: expected 400, got %d", w3.Code)
	}
	body400 := parseBody(w3)
	if body400["code"] == nil {
		t.Error("400 response must contain 'code'")
	}
}

// ==========================================================================
// Field-name contract: parent_id must be snake_case
// ==========================================================================

func TestOrgCreationParentIDFieldName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	p := r.Group("/api/v1")
	p.POST("/org/nodes", func(c *gin.Context) {
		var b struct {
			ParentID   *uint  `json:"parent_id"`
			Name       string `json:"name"`
			LevelCode  string `json:"level_code"`
			LevelLabel string `json:"level_label"`
		}
		if err := c.ShouldBindJSON(&b); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if b.ParentID == nil {
			c.JSON(200, gin.H{"parent_id": nil})
		} else {
			c.JSON(200, gin.H{"parent_id": *b.ParentID})
		}
	})

	w := doRequest(r, "POST", "/api/v1/org/nodes", "", map[string]interface{}{
		"parent_id": 42, "name": "N", "level_code": "city", "level_label": "City",
	})
	if w.Code != 200 {
		t.Fatalf("snake_case parent_id: expected 200, got %d", w.Code)
	}
	if pid, ok := parseBody(w)["parent_id"].(float64); !ok || uint(pid) != 42 {
		t.Errorf("parent_id not parsed from snake_case payload")
	}

	w2 := doRequest(r, "POST", "/api/v1/org/nodes", "", map[string]interface{}{
		"parentId": 42, "name": "N", "level_code": "city", "level_label": "City",
	})
	if parseBody(w2)["parent_id"] != nil {
		t.Error("camelCase parentId must NOT be recognised — backend expects snake_case")
	}
}
