package unit

import (
	"testing"

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
