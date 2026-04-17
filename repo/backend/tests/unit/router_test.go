package unit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"backend/internal/config"
	"backend/internal/router"
)

func TestSetupRouterReturnsEngine(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.GetConfig()
	r := router.SetupRouter(cfg, nil)
	if r == nil {
		t.Fatal("expected non-nil router")
	}
}

func TestHealthEndpointNoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.GetConfig()
	r := router.SetupRouter(cfg, nil)

	req, _ := http.NewRequest("GET", "/health", nil)
	req.RemoteAddr = "127.0.0.1:12345" // Needed for egress guard.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %v", body["status"])
	}
	if body["timestamp"] == nil || body["timestamp"] == "" {
		t.Error("expected non-empty timestamp")
	}
}

func TestProtectedRoutesReturn401WithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.GetConfig()
	r := router.SetupRouter(cfg, nil)

	protectedPaths := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/org/tree"},
		{"GET", "/api/v1/org/nodes"},
		{"GET", "/api/v1/master/sku"},
		{"GET", "/api/v1/analytics/kpis"},
		{"GET", "/api/v1/audit/logs"},
		{"GET", "/api/v1/security/keys"},
		{"GET", "/api/v1/integrations/endpoints"},
	}

	for _, tc := range protectedPaths {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req, _ := http.NewRequest(tc.method, tc.path, nil)
			req.RemoteAddr = "127.0.0.1:12345" // Needed for egress guard.
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401 for unauthenticated request, got %d", w.Code)
			}
		})
	}
}

func TestAllRouteGroupsRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.GetConfig()
	r := router.SetupRouter(cfg, nil)

	routes := r.Routes()
	routeSet := make(map[string]bool, len(routes))
	for _, rt := range routes {
		routeSet[rt.Method+" "+rt.Path] = true
	}

	// Verify key routes from each group are registered.
	expectedGroups := []struct {
		method string
		path   string
		group  string
	}{
		{"POST", "/api/v1/auth/login", "auth"},
		{"GET", "/api/v1/org/tree", "org"},
		{"GET", "/api/v1/master/:entity", "master"},
		{"GET", "/api/v1/versions/:entity", "versions"},
		{"GET", "/api/v1/ingestion/sources", "ingestion"},
		{"GET", "/api/v1/media", "media"},
		{"GET", "/api/v1/analytics/kpis", "analytics"},
		{"GET", "/api/v1/reports/schedules", "reports"},
		{"GET", "/api/v1/audit/logs", "audit"},
		{"GET", "/api/v1/security/keys", "security"},
		{"GET", "/api/v1/integrations/endpoints", "integrations"},
		{"GET", "/health", "health"},
	}

	for _, eg := range expectedGroups {
		key := eg.method + " " + eg.path
		if !routeSet[key] {
			t.Errorf("route group %q: expected %s to be registered", eg.group, key)
		}
	}
}

func TestCORSHeadersPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.GetConfig()
	r := router.SetupRouter(cfg, nil)

	req, _ := http.NewRequest("OPTIONS", "/api/v1/auth/login", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// CORS preflight should return 2xx or 204.
	if w.Code >= 400 {
		t.Errorf("expected successful CORS preflight, got %d", w.Code)
	}
}
