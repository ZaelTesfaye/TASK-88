package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	appErrors "backend/internal/errors"
	"backend/internal/config"
	"backend/internal/middleware"
	"backend/internal/router"
)

// setupErrorRouter creates a router with public endpoints that exercise the
// error framework.  No fakeAuthMiddleware — endpoints are unauthenticated.
func setupErrorRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(appErrors.ErrorHandlerMiddleware())

	r.GET("/test/not-found", func(c *gin.Context) {
		appErrors.RespondNotFound(c, "the requested resource was not found")
	})
	r.POST("/test/validate", func(c *gin.Context) {
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
	r.POST("/test/conflict", func(c *gin.Context) {
		appErrors.RespondConflict(c, "version activation conflict", gin.H{
			"entity": "sku", "version_id": 42,
		})
	})
	r.POST("/test/bad-request", func(c *gin.Context) {
		appErrors.RespondBadRequest(c, "invalid input", gin.H{"reason": "malformed JSON"})
	})
	r.GET("/test/internal-error", func(c *gin.Context) {
		appErrors.RespondInternalError(c, "unexpected failure")
	})

	return r
}

func TestNotFoundReturns404(t *testing.T) {
	r := setupErrorRouter()
	w := doRequest(r, "GET", "/test/not-found", "", nil)
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
	w := doRequest(r, "POST", "/test/validate", "", map[string]string{})
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
	w := doRequest(r, "POST", "/test/conflict", "", nil)
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

	cases := []struct {
		name   string
		method string
		path   string
		body   interface{}
		status int
	}{
		{"not found", "GET", "/test/not-found", nil, 404},
		{"validation", "POST", "/test/validate", map[string]string{}, 422},
		{"conflict", "POST", "/test/conflict", nil, 409},
		{"bad request", "POST", "/test/bad-request", nil, 400},
		{"internal", "GET", "/test/internal-error", nil, 500},
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

	// Test 401 and 403 against the real production router.
	t.Run("unauthorized via real router", func(t *testing.T) {
		cfg := config.GetConfig()
		rr := router.SetupRouter(cfg, nil)
		req, _ := http.NewRequest("GET", "/api/v1/org/tree", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Set("X-Correlation-ID", "unauth-test")
		w := httptest.NewRecorder()
		rr.ServeHTTP(w, req)
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
