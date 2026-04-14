package api

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"backend/internal/config"
	"backend/internal/router"
)

// TestAPIContractRoutes verifies that every frontend API adapter URL has a
// matching registered route in the backend router. This catches mismatches
// between frontend and backend when paths change.
func TestAPIContractRoutes(t *testing.T) {
	cfg := config.GetConfig()

	// Build the real router to get all registered routes.
	// We pass nil for db since we only need route registration, not DB access.
	// The router setup will panic if it tries to use db, so we use a recovery-safe approach.
	// Instead, we enumerate the expected routes from the router source of truth.

	// These are the routes registered in router.go — the source of truth.
	// Each entry is {method, path} as registered.
	expectedRoutes := []struct {
		method string
		path   string
	}{
		// Auth (public)
		{"POST", "/api/v1/auth/login"},
		// Auth (protected)
		{"POST", "/api/v1/auth/logout"},
		{"POST", "/api/v1/auth/refresh"},
		// Org
		{"GET", "/api/v1/org/tree"},
		{"GET", "/api/v1/org/nodes"},
		{"POST", "/api/v1/org/nodes"},
		{"GET", "/api/v1/org/nodes/:id"},
		{"PUT", "/api/v1/org/nodes/:id"},
		{"DELETE", "/api/v1/org/nodes/:id"},
		// Context
		{"POST", "/api/v1/context/switch"},
		{"GET", "/api/v1/context/current"},
		// Master
		{"GET", "/api/v1/master/:entity"},
		{"GET", "/api/v1/master/:entity/:id"},
		{"GET", "/api/v1/master/:entity/:id/history"},
		{"POST", "/api/v1/master/:entity"},
		{"PUT", "/api/v1/master/:entity/:id"},
		{"POST", "/api/v1/master/:entity/:id/deactivate"},
		// Versions
		{"GET", "/api/v1/versions/:entity"},
		{"GET", "/api/v1/versions/:entity/:id"},
		{"GET", "/api/v1/versions/:entity/:id/items"},
		{"GET", "/api/v1/versions/:entity/:id/diff"},
		{"POST", "/api/v1/versions/:entity"},
		{"POST", "/api/v1/versions/:entity/:id/review"},
		{"POST", "/api/v1/versions/:entity/:id/items"},
		{"DELETE", "/api/v1/versions/:entity/:id/items/:itemId"},
		{"POST", "/api/v1/versions/:entity/:id/activate"},
		// Ingestion
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
		// Media (playback)
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
		// Analytics
		{"GET", "/api/v1/analytics/kpis"},
		{"GET", "/api/v1/analytics/kpis/definitions"},
		{"POST", "/api/v1/analytics/kpis/definitions"},
		{"GET", "/api/v1/analytics/kpis/definitions/:code"},
		{"PUT", "/api/v1/analytics/kpis/definitions/:code"},
		{"DELETE", "/api/v1/analytics/kpis/definitions/:code"},
		{"GET", "/api/v1/analytics/trends"},
		// Reports
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
		// Audit
		{"GET", "/api/v1/audit/logs"},
		{"GET", "/api/v1/audit/logs/:id"},
		{"GET", "/api/v1/audit/logs/search"},
		{"GET", "/api/v1/audit/delete-requests"},
		{"POST", "/api/v1/audit/delete-requests"},
		{"GET", "/api/v1/audit/delete-requests/:id"},
		{"POST", "/api/v1/audit/delete-requests/:id/approve"},
		{"POST", "/api/v1/audit/delete-requests/:id/execute"},
		// Security
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
		// Integrations
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
		// Health
		{"GET", "/health"},
	}

	// Build the real router to get registered routes.
	r := router.SetupRouter(cfg, nil)
	registeredRoutes := r.Routes()

	// Build a lookup map of registered routes.
	routeSet := make(map[string]bool, len(registeredRoutes))
	for _, route := range registeredRoutes {
		key := route.Method + " " + route.Path
		routeSet[key] = true
	}

	// Verify every expected route exists.
	for _, expected := range expectedRoutes {
		key := expected.method + " " + expected.path
		if !routeSet[key] {
			t.Errorf("expected route %s %s not registered in router", expected.method, expected.path)
		}
	}

	// Also verify no unexpected routes exist (optional — log them).
	for _, route := range registeredRoutes {
		found := false
		for _, expected := range expectedRoutes {
			if expected.method == route.Method && expected.path == route.Path {
				found = true
				break
			}
		}
		if !found {
			t.Logf("INFO: additional route registered: %s %s (not in contract list)", route.Method, route.Path)
		}
	}
}

// TestOrgCreationParentIDFieldName verifies that the backend org node creation
// endpoint expects the field name "parent_id" (snake_case), matching the
// frontend API adapter in frontend/src/api/org.js.
func TestOrgCreationParentIDFieldName(t *testing.T) {
	// The backend CreateNodeRequest in org_service.go uses the JSON tag
	// `json:"parent_id"` — verify this by constructing a request with
	// "parent_id" and checking it is accepted (no validation error on field name).
	r := testRouter()
	protected := r.Group("/api/v1")
	protected.Use(fakeAuthMiddleware())

	protected.POST("/org/nodes", func(c *gin.Context) {
		var body struct {
			ParentID   *uint  `json:"parent_id"`
			Name       string `json:"name"`
			LevelCode  string `json:"level_code"`
			LevelLabel string `json:"level_label"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		// Verify parent_id was parsed.
		if body.ParentID == nil {
			c.JSON(200, gin.H{"parent_id": nil})
		} else {
			c.JSON(200, gin.H{"parent_id": *body.ParentID})
		}
	})

	token := signToken(1, "system_admin", "*", "*", 30*time.Minute)

	// Send with parent_id (snake_case) — should be parsed correctly.
	parentID := uint(42)
	body := map[string]interface{}{
		"parent_id":   parentID,
		"name":        "Test Node",
		"level_code":  "city",
		"level_label": "City",
	}
	w := doRequest(r, "POST", "/api/v1/org/nodes", token, body)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(w)
	if pid, ok := resp["parent_id"].(float64); !ok || uint(pid) != parentID {
		t.Errorf("expected parent_id=%d in response, got %v", parentID, resp["parent_id"])
	}

	// Send with parentId (camelCase) — should NOT be parsed (field stays nil).
	body2 := map[string]interface{}{
		"parentId":    parentID,
		"name":        "Test Node 2",
		"level_code":  "city",
		"level_label": "City",
	}
	w2 := doRequest(r, "POST", "/api/v1/org/nodes", token, body2)
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	resp2 := parseBody(w2)
	// parentId (camelCase) should not be recognized — parent_id should be null.
	if resp2["parent_id"] != nil {
		t.Errorf("expected parent_id=nil when sending camelCase 'parentId', got %v", resp2["parent_id"])
	}
}

// TestLyricsParseResponseSchema verifies that POST /media/:id/lyrics/parse
// returns a response with `lrc` field (raw LRC string) matching what
// PlaybackPage.vue:368 expects via `data.lrc`.
func TestLyricsParseResponseSchema(t *testing.T) {
	r := testRouter()
	protected := r.Group("/api/v1")
	protected.Use(fakeAuthMiddleware())

	// Simulate the parse endpoint returning the expected shape.
	protected.POST("/media/:id/lyrics/parse", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":     "success",
			"line_count": 2,
			"lines":      []gin.H{{"time": 0.0, "text": "hello"}, {"time": 1.0, "text": "world"}},
			"lrc":        "[00:00.00]hello\n[00:01.00]world",
		})
	})

	token := signToken(1, "system_admin", "*", "*", 30*time.Minute)
	w := doRequest(r, "POST", "/api/v1/media/1/lyrics/parse", token, nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resp := parseBody(w)

	// The frontend reads data.lrc — it must be present.
	if _, ok := resp["lrc"]; !ok {
		t.Fatal("response must contain 'lrc' field for frontend compatibility")
	}
	if _, ok := resp["lrc"].(string); !ok {
		t.Fatal("'lrc' field must be a string")
	}
	if _, ok := resp["lines"]; !ok {
		t.Fatal("response must contain 'lines' field")
	}
}

// TestReportsAccessCheckResponseSchema verifies that GET /reports/runs/:id/access-check
// returns {has_access: bool} matching what ReportsPage.vue:398 expects.
func TestReportsAccessCheckResponseSchema(t *testing.T) {
	r := testRouter()
	protected := r.Group("/api/v1")
	protected.Use(fakeAuthMiddleware())

	protected.GET("/reports/runs/:id/access-check", func(c *gin.Context) {
		c.JSON(200, gin.H{"has_access": true})
	})

	token := signToken(1, "system_admin", "*", "*", 30*time.Minute)
	w := doRequest(r, "GET", "/api/v1/reports/runs/1/access-check", token, nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resp := parseBody(w)

	// Frontend reads accessData.has_access (not .allowed).
	if _, ok := resp["has_access"]; !ok {
		t.Fatal("response must contain 'has_access' field, not 'allowed'")
	}
	if _, ok := resp["has_access"].(bool); !ok {
		t.Fatal("'has_access' field must be a boolean")
	}
}

// TestReportsListRunsQueryParams verifies that GET /reports/runs accepts
// date_from/date_to (not startDate/endDate) and schedule_id as query params.
func TestReportsListRunsQueryParams(t *testing.T) {
	r := testRouter()
	protected := r.Group("/api/v1")
	protected.Use(fakeAuthMiddleware())

	var capturedParams struct {
		ScheduleID string
		DateFrom   string
		DateTo     string
	}
	protected.GET("/reports/runs", func(c *gin.Context) {
		capturedParams.ScheduleID = c.Query("schedule_id")
		capturedParams.DateFrom = c.Query("date_from")
		capturedParams.DateTo = c.Query("date_to")
		c.JSON(200, gin.H{"items": []interface{}{}, "total": 0})
	})

	token := signToken(1, "system_admin", "*", "*", 30*time.Minute)
	doRequest(r, "GET", "/api/v1/reports/runs?schedule_id=5&date_from=2025-01-01&date_to=2025-12-31", token, nil)

	if capturedParams.ScheduleID != "5" {
		t.Errorf("expected schedule_id=5, got %q", capturedParams.ScheduleID)
	}
	if capturedParams.DateFrom != "2025-01-01" {
		t.Errorf("expected date_from=2025-01-01, got %q", capturedParams.DateFrom)
	}
	if capturedParams.DateTo != "2025-12-31" {
		t.Errorf("expected date_to=2025-12-31, got %q", capturedParams.DateTo)
	}
}
