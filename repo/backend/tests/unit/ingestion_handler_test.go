package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"backend/internal/handlers"
	"backend/internal/middleware"
)

// Ingestion handlers use fmt.Sscanf for ID parsing (not strconv.ParseUint),
// so invalid IDs become 0 and proceed to DB queries. With nil DB these panic.
// Unit tests here cover what's testable without a DB: handler construction,
// route wiring, and the panic-recovery behavior.

func ingestionRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.RecoveryMiddleware())
	h := handlers.NewIngestionHandler(nil)
	g := r.Group("/api/v1/ingestion")
	g.POST("/sources", h.CreateSource)
	g.GET("/sources/:id", h.GetSource)
	g.GET("/jobs/:id", h.GetJob)
	g.POST("/jobs/:id/retry", h.RetryJob)
	g.GET("/jobs/:id/checkpoints", h.ListCheckpoints)
	g.GET("/jobs/:id/failures", h.ListFailures)
	return r
}

func TestIngestionHandlerConstructor(t *testing.T) {
	h := handlers.NewIngestionHandler(nil)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestIngestionHandlerNilDBRecovery(t *testing.T) {
	r := ingestionRouter()
	// With nil DB and recovery middleware, requests should get 500 (not crash).
	req, _ := http.NewRequest("GET", "/api/v1/ingestion/sources/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Expect 500 (panic recovered) or 401 (auth check first).
	if w.Code != http.StatusInternalServerError && w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 500/401 with nil DB, got %d", w.Code)
	}
}

func TestIngestionHandlerJobCheckpointsNilDBRecovery(t *testing.T) {
	r := ingestionRouter()
	req, _ := http.NewRequest("GET", "/api/v1/ingestion/jobs/1/checkpoints", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError && w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 500/401, got %d", w.Code)
	}
}

func TestIngestionHandlerJobFailuresNilDBRecovery(t *testing.T) {
	r := ingestionRouter()
	req, _ := http.NewRequest("GET", "/api/v1/ingestion/jobs/1/failures", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError && w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 500/401, got %d", w.Code)
	}
}
