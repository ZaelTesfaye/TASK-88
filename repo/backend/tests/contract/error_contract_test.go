package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	appErrors "backend/internal/errors"
	"backend/internal/rbac"
)

// setupErrorRouter creates a router with endpoints that return various error codes.
func setupErrorRouter() *gin.Engine {
	r := testRouter()
	protected := r.Group("/api/v1")
	protected.Use(fakeAuthMiddleware())

	protected.GET("/not-found", func(c *gin.Context) {
		appErrors.RespondNotFound(c, "the requested resource was not found")
	})

	protected.POST("/validate", func(c *gin.Context) {
		var body struct {
			Name  string `json:"name" binding:"required"`
			Email string `json:"email" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			appErrors.RespondValidationError(c, "validation failed", gin.H{
				"fields": []gin.H{
					{"field": "name", "message": "name is required"},
					{"field": "email", "message": "email is required"},
				},
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": body})
	})

	protected.POST("/conflict", func(c *gin.Context) {
		appErrors.RespondConflict(c, "version activation conflict", gin.H{
			"entity": "sku", "version_id": 42,
		})
	})

	protected.POST("/bad-request", func(c *gin.Context) {
		appErrors.RespondBadRequest(c, "invalid input", gin.H{"reason": "malformed JSON"})
	})

	protected.GET("/internal-error", func(c *gin.Context) {
		appErrors.RespondInternalError(c, "unexpected failure")
	})

	adminOnly := protected.Group("/admin-only")
	adminOnly.Use(rbac.RequireRole(rbac.SystemAdmin))
	adminOnly.GET("/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin resource"})
	})

	return r
}

func TestNotFoundReturns404(t *testing.T) {
	r := setupErrorRouter()
	tok := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)
	w := doRequest(r, "GET", "/api/v1/not-found", tok, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	resp := parseBody(w)
	if resp["code"] != "NOT_FOUND" {
		t.Errorf("expected code=NOT_FOUND, got %v", resp["code"])
	}
	if resp["message"] == nil || resp["message"] == "" {
		t.Error("missing 'message'")
	}
	if resp["correlationId"] == nil || resp["correlationId"] == "" {
		t.Error("missing 'correlationId'")
	}
}

func TestValidationReturns422(t *testing.T) {
	r := setupErrorRouter()
	tok := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)
	w := doRequest(r, "POST", "/api/v1/validate", tok, map[string]string{})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
	resp := parseBody(w)
	if resp["code"] != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %v", resp["code"])
	}
	details, _ := resp["details"].(map[string]interface{})
	fields, _ := details["fields"].([]interface{})
	if len(fields) == 0 {
		t.Fatal("fields array should not be empty")
	}
}

func TestConflictReturns409(t *testing.T) {
	r := setupErrorRouter()
	tok := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)
	w := doRequest(r, "POST", "/api/v1/conflict", tok, nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
	resp := parseBody(w)
	if resp["code"] != "CONFLICT" {
		t.Errorf("expected CONFLICT, got %v", resp["code"])
	}
}

func TestUnifiedErrorContract(t *testing.T) {
	r := setupErrorRouter()
	tok := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)

	cases := []struct {
		name   string
		method string
		path   string
		body   interface{}
		status int
	}{
		{"not found", "GET", "/api/v1/not-found", nil, 404},
		{"validation", "POST", "/api/v1/validate", map[string]string{}, 422},
		{"conflict", "POST", "/api/v1/conflict", nil, 409},
		{"bad request", "POST", "/api/v1/bad-request", nil, 400},
		{"internal", "GET", "/api/v1/internal-error", nil, 500},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf *bytes.Buffer
			if tc.body != nil {
				b, _ := json.Marshal(tc.body)
				buf = bytes.NewBuffer(b)
			} else {
				buf = bytes.NewBuffer(nil)
			}
			req, _ := http.NewRequest(tc.method, tc.path, buf)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+tok)
			req.Header.Set("X-Correlation-ID", "contract-test")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("expected %d, got %d: %s", tc.status, w.Code, w.Body.String())
			}

			resp := parseBody(w)
			for _, field := range []string{"code", "message", "correlationId"} {
				if resp[field] == nil || resp[field] == "" {
					t.Errorf("missing '%s' in error response", field)
				}
			}
		})
	}

	t.Run("forbidden", func(t *testing.T) {
		nonAdmin := signToken(99, rbac.StandardUser, "*", "*", 30*time.Minute)
		req, _ := http.NewRequest("GET", "/api/v1/admin-only/resource", nil)
		req.Header.Set("Authorization", "Bearer "+nonAdmin)
		req.Header.Set("X-Correlation-ID", "forbidden-test")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
		resp := parseBody(w)
		for _, f := range []string{"code", "message", "correlationId"} {
			if resp[f] == nil || resp[f] == "" {
				t.Errorf("403 missing '%s'", f)
			}
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/not-found", nil)
		req.Header.Set("X-Correlation-ID", "unauth-test")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
		resp := parseBody(w)
		for _, f := range []string{"code", "message", "correlationId"} {
			if resp[f] == nil || resp[f] == "" {
				t.Errorf("401 missing '%s'", f)
			}
		}
	})
}
