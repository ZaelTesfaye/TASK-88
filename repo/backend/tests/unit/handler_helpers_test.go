package unit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	appErrors "backend/internal/errors"
)

// TestHandlerServiceErrorMapping verifies that different AppError codes map
// to the correct HTTP status codes when processed through the error middleware
// and a handler that mimics handleServiceError behavior.
func TestHandlerServiceErrorMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name       string
		errorCode  string
		httpStatus int
	}{
		{"BadRequest", "BAD_REQUEST", http.StatusBadRequest},
		{"NotFound", "NOT_FOUND", http.StatusNotFound},
		{"Conflict", "CONFLICT", http.StatusConflict},
		{"ValidationError", "VALIDATION_ERROR", http.StatusUnprocessableEntity},
		{"Forbidden", "FORBIDDEN", http.StatusForbidden},
		{"InternalError", "INTERNAL_ERROR", http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(appErrors.ErrorHandlerMiddleware())

			r.GET("/test", func(c *gin.Context) {
				appErr := &appErrors.AppError{Code: tc.errorCode, Message: "test error"}
				switch appErr.Code {
				case "BAD_REQUEST":
					appErrors.RespondWithError(c, http.StatusBadRequest, appErr)
				case "NOT_FOUND":
					appErrors.RespondWithError(c, http.StatusNotFound, appErr)
				case "CONFLICT":
					appErrors.RespondWithError(c, http.StatusConflict, appErr)
				case "VALIDATION_ERROR":
					appErrors.RespondWithError(c, http.StatusUnprocessableEntity, appErr)
				case "FORBIDDEN":
					appErrors.RespondWithError(c, http.StatusForbidden, appErr)
				default:
					appErrors.RespondWithError(c, http.StatusInternalServerError, appErr)
				}
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.httpStatus {
				t.Errorf("expected %d, got %d", tc.httpStatus, w.Code)
			}

			var body map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &body)
			if body["code"] != tc.errorCode {
				t.Errorf("expected code %q, got %v", tc.errorCode, body["code"])
			}
		})
	}
}

// TestInvalidIDParamReturns400 verifies that a non-numeric ID param results
// in a 400 response, matching the pattern in handler parseIDParam.
func TestInvalidIDParamReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(appErrors.ErrorHandlerMiddleware())

	r.GET("/test/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		var id uint
		if _, err := parseUint(idStr); err != nil {
			appErrors.RespondBadRequest(c, "invalid id parameter: "+idStr, nil)
			return
		}
		_ = id
		c.JSON(200, gin.H{"ok": true})
	})

	tests := []struct {
		path   string
		expect int
	}{
		{"/test/42", 200},
		{"/test/abc", 400},
		{"/test/-1", 400},
		{"/test/3.14", 400},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tc.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.expect {
				t.Errorf("path %s: expected %d, got %d", tc.path, tc.expect, w.Code)
			}
		})
	}
}

func parseUint(s string) (uint, error) {
	var n uint
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, appErrors.BadRequest("invalid id", nil)
		}
		n = n*10 + uint(ch-'0')
	}
	return n, nil
}
