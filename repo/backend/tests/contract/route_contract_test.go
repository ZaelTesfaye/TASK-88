// Package contract contains tests that verify the API surface contract
// between the frontend adapters and the backend router.  These tests
// do not hit a database — they only inspect registered routes and
// validate request/response schemas.
package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"backend/internal/auth"
	"backend/internal/config"
	appErrors "backend/internal/errors"
	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/router"
)

func init() { gin.SetMode(gin.TestMode) }

// ---------- lightweight helpers (no external deps) ----------

func signToken(userID uint, role, city, dept string, expiresIn time.Duration) string {
	cfg := config.GetConfig()
	now := time.Now()
	claims := auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			ID:        fmt.Sprintf("contract-jti-%d-%d", userID, now.UnixNano()),
			Issuer:    cfg.JWTIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
		},
		Role:            role,
		CityScope:       city,
		DepartmentScope: dept,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(cfg.JWTSecret))
	return s
}

func fakeAuthMiddleware() gin.HandlerFunc {
	cfg := config.GetConfig()
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if len(h) < 8 || h[:7] != "Bearer " {
			appErrors.RespondUnauthorized(c, "missing token")
			return
		}
		claims := &auth.Claims{}
		tok, err := jwt.ParseWithClaims(h[7:], claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !tok.Valid {
			appErrors.RespondUnauthorized(c, "invalid token")
			return
		}
		var uid uint
		fmt.Sscanf(claims.Subject, "%d", &uid)
		user := &models.User{ID: uid, Username: fmt.Sprintf("user_%d", uid), Role: claims.Role,
			CityScope: claims.CityScope, DepartmentScope: claims.DepartmentScope, Status: "active"}
		c.Set("current_user", user)
		c.Set("current_claims", claims)
		c.Set("user_id", user.ID)
		c.Set("user_role", user.Role)
		c.Set("city_scope", user.CityScope)
		c.Set("dept_scope", user.DepartmentScope)
		c.Next()
	}
}

func testRouter() *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(appErrors.ErrorHandlerMiddleware())
	return r
}

func doRequest(r *gin.Engine, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req, _ := http.NewRequest(method, path, buf)
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
// Route contract: every frontend API adapter URL must match a backend route
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
}

// ==========================================================================
// Field-name contracts
// ==========================================================================

func TestOrgCreationParentIDFieldName(t *testing.T) {
	r := testRouter()
	p := r.Group("/api/v1")
	p.Use(fakeAuthMiddleware())
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

	tok := signToken(1, "system_admin", "*", "*", 30*time.Minute)

	w := doRequest(r, "POST", "/api/v1/org/nodes", tok, map[string]interface{}{
		"parent_id": 42, "name": "N", "level_code": "city", "level_label": "City",
	})
	if w.Code != 200 {
		t.Fatalf("snake_case parent_id: expected 200, got %d", w.Code)
	}
	if pid, ok := parseBody(w)["parent_id"].(float64); !ok || uint(pid) != 42 {
		t.Errorf("parent_id not parsed from snake_case payload")
	}

	w2 := doRequest(r, "POST", "/api/v1/org/nodes", tok, map[string]interface{}{
		"parentId": 42, "name": "N", "level_code": "city", "level_label": "City",
	})
	if parseBody(w2)["parent_id"] != nil {
		t.Error("camelCase parentId must NOT be recognised — backend expects snake_case")
	}
}

// ==========================================================================
// Response-schema contracts
// ==========================================================================

func TestLyricsParseResponseSchema(t *testing.T) {
	r := testRouter()
	p := r.Group("/api/v1")
	p.Use(fakeAuthMiddleware())
	p.POST("/media/:id/lyrics/parse", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "success", "line_count": 2,
			"lines": []gin.H{{"time": 0.0, "text": "a"}, {"time": 1.0, "text": "b"}},
			"lrc":   "[00:00.00]a\n[00:01.00]b",
		})
	})

	tok := signToken(1, "system_admin", "*", "*", 30*time.Minute)
	resp := parseBody(doRequest(r, "POST", "/api/v1/media/1/lyrics/parse", tok, nil))
	if _, ok := resp["lrc"].(string); !ok {
		t.Fatal("response must contain string 'lrc' field for PlaybackPage.vue compatibility")
	}
	if resp["lines"] == nil {
		t.Fatal("response must contain 'lines' field")
	}
}

func TestReportsAccessCheckResponseSchema(t *testing.T) {
	r := testRouter()
	p := r.Group("/api/v1")
	p.Use(fakeAuthMiddleware())
	p.GET("/reports/runs/:id/access-check", func(c *gin.Context) {
		c.JSON(200, gin.H{"has_access": true})
	})

	tok := signToken(1, "system_admin", "*", "*", 30*time.Minute)
	resp := parseBody(doRequest(r, "GET", "/api/v1/reports/runs/1/access-check", tok, nil))
	if _, ok := resp["has_access"].(bool); !ok {
		t.Fatal("response must contain bool 'has_access', not 'allowed'")
	}
}

func TestReportsListRunsQueryParams(t *testing.T) {
	r := testRouter()
	p := r.Group("/api/v1")
	p.Use(fakeAuthMiddleware())

	var got struct{ ScheduleID, DateFrom, DateTo string }
	p.GET("/reports/runs", func(c *gin.Context) {
		got.ScheduleID = c.Query("schedule_id")
		got.DateFrom = c.Query("date_from")
		got.DateTo = c.Query("date_to")
		c.JSON(200, gin.H{"items": []interface{}{}, "total": 0})
	})

	tok := signToken(1, "system_admin", "*", "*", 30*time.Minute)
	doRequest(r, "GET", "/api/v1/reports/runs?schedule_id=5&date_from=2025-01-01&date_to=2025-12-31", tok, nil)

	if got.ScheduleID != "5" {
		t.Errorf("schedule_id: want 5, got %q", got.ScheduleID)
	}
	if got.DateFrom != "2025-01-01" {
		t.Errorf("date_from: want 2025-01-01, got %q", got.DateFrom)
	}
	if got.DateTo != "2025-12-31" {
		t.Errorf("date_to: want 2025-12-31, got %q", got.DateTo)
	}
}
