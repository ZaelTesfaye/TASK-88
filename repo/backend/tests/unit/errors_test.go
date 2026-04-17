package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	appErrors "backend/internal/errors"
)

func TestAppErrorErrorMethod(t *testing.T) {
	err := appErrors.NewAppError("TEST_CODE", "something went wrong", nil)
	got := err.Error()
	if got != "[TEST_CODE] something went wrong" {
		t.Errorf("Error() = %q, want %q", got, "[TEST_CODE] something went wrong")
	}
}

func TestAppErrorWithDetails(t *testing.T) {
	details := map[string]string{"field": "name"}
	err := appErrors.NewAppError("VALIDATION_ERROR", "invalid", details)
	if err.Details == nil {
		t.Error("expected Details to be set")
	}
	if err.Code != "VALIDATION_ERROR" {
		t.Errorf("Code = %q, want VALIDATION_ERROR", err.Code)
	}
}

func TestRespondForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("correlation_id", "test-cid")

	appErrors.RespondForbidden(c, "access denied")

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}
