package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"backend/internal/handlers"
	"backend/internal/middleware"
)

func masterRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	h := handlers.NewMasterHandler(nil)
	g := r.Group("/api/v1/master")
	g.GET("/:entity/:id", h.GetRecord)
	g.GET("/:entity/:id/history", h.GetRecordHistory)
	g.POST("/:entity/:id/deactivate", h.DeactivateRecord)
	return r
}

func TestMasterHandlerGetRecordInvalidID(t *testing.T) {
	r := masterRouter()
	req, _ := http.NewRequest("GET", "/api/v1/master/sku/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMasterHandlerHistoryInvalidID(t *testing.T) {
	r := masterRouter()
	req, _ := http.NewRequest("GET", "/api/v1/master/sku/xyz/history", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMasterHandlerDeactivateInvalidID(t *testing.T) {
	r := masterRouter()
	req, _ := http.NewRequest("POST", "/api/v1/master/sku/bad/deactivate", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMasterHandlerDeactivateMissingReason(t *testing.T) {
	r := masterRouter()
	req, _ := http.NewRequest("POST", "/api/v1/master/sku/1/deactivate", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing reason, got %d: %s", w.Code, w.Body.String())
	}
}
