package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	appErrors "backend/internal/errors"
	"backend/internal/middleware"
	"backend/internal/rbac"
)

// setupErrorRouter creates a router with endpoints that return various error codes.
func setupErrorRouter() *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(appErrors.ErrorHandlerMiddleware())

	protected := r.Group("/api/v1")
	protected.Use(fakeAuthMiddleware())

	// 404 Not Found
	protected.GET("/not-found", func(c *gin.Context) {
		appErrors.RespondNotFound(c, "the requested resource was not found")
	})

	// 422 Validation Error
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

	// 409 Conflict
	protected.POST("/conflict", func(c *gin.Context) {
		appErrors.RespondConflict(c, "version activation conflict", gin.H{
			"entity":     "sku",
			"version_id": 42,
		})
	})

	// 400 Bad Request
	protected.POST("/bad-request", func(c *gin.Context) {
		appErrors.RespondBadRequest(c, "invalid input", gin.H{"reason": "malformed JSON"})
	})

	// 500 Internal Error
	protected.GET("/internal-error", func(c *gin.Context) {
		appErrors.RespondInternalError(c, "unexpected failure")
	})

	// 403 Forbidden (role check)
	adminOnly := protected.Group("/admin-only")
	adminOnly.Use(rbac.RequireRole(rbac.SystemAdmin))
	adminOnly.GET("/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin resource"})
	})

	return r
}

// ---------- tests ----------

func TestNotFoundReturns404(t *testing.T) {
	r := setupErrorRouter()
	token := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)

	w := doRequest(r, "GET", "/api/v1/not-found", token, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(w)

	// Verify error contract fields.
	code, ok := resp["code"].(string)
	if !ok || code == "" {
		t.Error("error response must have 'code' field")
	}
	if code != "NOT_FOUND" {
		t.Errorf("expected code=NOT_FOUND, got %s", code)
	}

	message, ok := resp["message"].(string)
	if !ok || message == "" {
		t.Error("error response must have 'message' field")
	}

	// correlationId should be present (set by RequestIDMiddleware).
	corrID, ok := resp["correlationId"].(string)
	if !ok || corrID == "" {
		t.Error("error response must have 'correlationId' field")
	}
}

func TestValidationReturns422(t *testing.T) {
	r := setupErrorRouter()
	token := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)

	// Send empty body to trigger validation error.
	w := doRequest(r, "POST", "/api/v1/validate", token, map[string]string{})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(w)

	code := resp["code"].(string)
	if code != "VALIDATION_ERROR" {
		t.Errorf("expected code=VALIDATION_ERROR, got %s", code)
	}

	// Details should contain field-level errors.
	details, ok := resp["details"].(map[string]interface{})
	if !ok {
		t.Fatal("validation error should have 'details' with field-level errors")
	}
	fields, ok := details["fields"].([]interface{})
	if !ok {
		t.Fatal("details should contain 'fields' array")
	}
	if len(fields) == 0 {
		t.Error("fields array should not be empty")
	}

	// Each field error should have 'field' and 'message'.
	for _, f := range fields {
		fieldMap, ok := f.(map[string]interface{})
		if !ok {
			t.Error("each field error should be a map")
			continue
		}
		if fieldMap["field"] == nil || fieldMap["field"] == "" {
			t.Error("field error missing 'field' key")
		}
		if fieldMap["message"] == nil || fieldMap["message"] == "" {
			t.Error("field error missing 'message' key")
		}
	}

	// correlationId must be present.
	if resp["correlationId"] == nil || resp["correlationId"] == "" {
		t.Error("validation error response must have correlationId")
	}
}

func TestConflictReturns409(t *testing.T) {
	r := setupErrorRouter()
	token := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)

	w := doRequest(r, "POST", "/api/v1/conflict", token, nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(w)

	code := resp["code"].(string)
	if code != "CONFLICT" {
		t.Errorf("expected code=CONFLICT, got %s", code)
	}

	// Details should contain conflict information.
	details, ok := resp["details"].(map[string]interface{})
	if !ok {
		t.Fatal("conflict error should have 'details'")
	}
	if details["entity"] != "sku" {
		t.Errorf("expected entity=sku in details, got %v", details["entity"])
	}

	// correlationId must be present.
	if resp["correlationId"] == nil || resp["correlationId"] == "" {
		t.Error("conflict error response must have correlationId")
	}
}

func TestUnifiedErrorContract(t *testing.T) {
	r := setupErrorRouter()
	token := signToken(1, rbac.SystemAdmin, "*", "*", 30*time.Minute)

	// Test that ALL error responses follow the same contract: code, message, correlationId.
	errorEndpoints := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
	}{
		{"not found", "GET", "/api/v1/not-found", nil, http.StatusNotFound},
		{"validation error", "POST", "/api/v1/validate", map[string]string{}, http.StatusUnprocessableEntity},
		{"conflict", "POST", "/api/v1/conflict", nil, http.StatusConflict},
		{"bad request", "POST", "/api/v1/bad-request", nil, http.StatusBadRequest},
		{"internal error", "GET", "/api/v1/internal-error", nil, http.StatusInternalServerError},
	}

	for _, tc := range errorEndpoints {
		t.Run(tc.name, func(t *testing.T) {
			var reqBody *bytes.Buffer
			if tc.body != nil {
				b, _ := json.Marshal(tc.body)
				reqBody = bytes.NewBuffer(b)
			} else {
				reqBody = bytes.NewBuffer(nil)
			}

			req, _ := http.NewRequest(tc.method, tc.path, reqBody)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Correlation-ID", "unified-test-123")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Fatalf("expected %d, got %d: %s", tc.expectedStatus, w.Code, w.Body.String())
			}

			resp := parseBody(w)

			// Every error response must have these three fields.
			if resp["code"] == nil || resp["code"] == "" {
				t.Errorf("%s: missing 'code' field", tc.name)
			}
			if resp["message"] == nil || resp["message"] == "" {
				t.Errorf("%s: missing 'message' field", tc.name)
			}
			if resp["correlationId"] == nil || resp["correlationId"] == "" {
				t.Errorf("%s: missing 'correlationId' field", tc.name)
			}

			// Verify correlationId matches what we sent.
			corrID := resp["correlationId"].(string)
			if corrID != "unified-test-123" {
				t.Errorf("%s: expected correlationId='unified-test-123', got %q", tc.name, corrID)
			}

			// Verify X-Correlation-ID header is echoed.
			headerCorrID := w.Header().Get("X-Correlation-ID")
			if headerCorrID != "unified-test-123" {
				t.Errorf("%s: expected X-Correlation-ID header='unified-test-123', got %q", tc.name, headerCorrID)
			}
		})
	}

	// Test that 403 Forbidden also follows the contract.
	t.Run("forbidden", func(t *testing.T) {
		// Use a non-admin token to trigger 403.
		nonAdminToken := signToken(99, rbac.StandardUser, "*", "*", 30*time.Minute)

		req, _ := http.NewRequest("GET", "/api/v1/admin-only/resource", nil)
		req.Header.Set("Authorization", "Bearer "+nonAdminToken)
		req.Header.Set("X-Correlation-ID", "forbidden-test")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}

		resp := parseBody(w)
		if resp["code"] == nil || resp["code"] == "" {
			t.Error("403 response missing 'code'")
		}
		if resp["message"] == nil || resp["message"] == "" {
			t.Error("403 response missing 'message'")
		}
		if resp["correlationId"] == nil || resp["correlationId"] == "" {
			t.Error("403 response missing 'correlationId'")
		}
	})

	// Test that 401 Unauthorized follows the contract.
	t.Run("unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/not-found", nil)
		// No Authorization header.
		req.Header.Set("X-Correlation-ID", "unauth-test")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}

		resp := parseBody(w)
		if resp["code"] == nil || resp["code"] == "" {
			t.Error("401 response missing 'code'")
		}
		if resp["message"] == nil || resp["message"] == "" {
			t.Error("401 response missing 'message'")
		}
		if resp["correlationId"] == nil || resp["correlationId"] == "" {
			t.Error("401 response missing 'correlationId'")
		}
	})
}
