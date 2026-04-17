package unit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"backend/internal/handlers"
	"backend/internal/middleware"
)

// These tests exercise the real production ReportHandler code.
// Input validation paths don't require a DB.

func reportRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())

	h := handlers.NewReportHandler(nil) // nil DB — input validation only
	g := r.Group("/api/v1/reports")
	{
		g.POST("/schedules", h.CreateSchedule)
		g.GET("/schedules/:id", h.GetSchedule)
		g.PATCH("/schedules/:id", h.UpdateSchedule)
		g.DELETE("/schedules/:id", h.DeleteSchedule)
		g.POST("/schedules/:id/trigger", h.TriggerSchedule)
		g.GET("/runs/:id/download", h.DownloadRun)
		g.GET("/runs/:id/access-check", h.AccessCheck)
	}
	return r
}

func TestReportHandlerCreateScheduleMissingFields(t *testing.T) {
	r := reportRouter()

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{"empty body", map[string]interface{}{}},
		{"missing name", map[string]interface{}{"cron_expr": "0 8 * * 1"}},
		{"missing cron_expr", map[string]interface{}{"name": "Weekly"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(tc.body)
			req, _ := http.NewRequest("POST", "/api/v1/reports/schedules", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusUnprocessableEntity && w.Code != http.StatusBadRequest {
				t.Errorf("expected 422 or 400, got %d: %s", w.Code, w.Body.String())
			}
			var resp map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			if resp["message"] == nil {
				t.Error("expected validation error message")
			}
		})
	}
}

func TestReportHandlerGetScheduleInvalidID(t *testing.T) {
	r := reportRouter()
	req, _ := http.NewRequest("GET", "/api/v1/reports/schedules/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["message"] == nil {
		t.Error("expected error message for invalid ID")
	}
}

func TestReportHandlerTriggerInvalidID(t *testing.T) {
	r := reportRouter()
	req, _ := http.NewRequest("POST", "/api/v1/reports/schedules/xyz/trigger", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestReportHandlerDownloadInvalidID(t *testing.T) {
	r := reportRouter()
	req, _ := http.NewRequest("GET", "/api/v1/reports/runs/nope/download", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestReportHandlerAccessCheckInvalidID(t *testing.T) {
	r := reportRouter()
	req, _ := http.NewRequest("GET", "/api/v1/reports/runs/bad/access-check", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestReportHandlerDeleteInvalidID(t *testing.T) {
	r := reportRouter()
	req, _ := http.NewRequest("DELETE", "/api/v1/reports/schedules/foo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
