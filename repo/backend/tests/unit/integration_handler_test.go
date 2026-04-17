package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"backend/internal/handlers"
	"backend/internal/middleware"
)

// Integration handlers use fmt.Sscanf for ID parsing. Invalid IDs become 0 and
// proceed to DB queries. Unit tests cover handler construction and panic recovery.

func integrationRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.RecoveryMiddleware())
	h := handlers.NewIntegrationHandler(nil)
	g := r.Group("/api/v1/integrations")
	g.POST("/endpoints", h.CreateEndpoint)
	g.POST("/endpoints/:id/test", h.TestEndpoint)
	g.GET("/endpoints/:id", h.GetEndpoint)
	g.POST("/connectors", h.CreateConnector)
	g.POST("/connectors/:id/health-check", h.HealthCheckConnector)
	g.GET("/deliveries/:id", h.GetDelivery)
	g.POST("/deliveries/:id/retry", h.RetryDelivery)
	return r
}

func TestIntegrationHandlerConstructor(t *testing.T) {
	h := handlers.NewIntegrationHandler(nil)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestIntegrationHandlerGetEndpointNilDB(t *testing.T) {
	r := integrationRouter()
	req, _ := http.NewRequest("GET", "/api/v1/integrations/endpoints/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError && w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 500/401 with nil DB, got %d", w.Code)
	}
}

func TestIntegrationHandlerGetDeliveryNilDB(t *testing.T) {
	r := integrationRouter()
	req, _ := http.NewRequest("GET", "/api/v1/integrations/deliveries/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError && w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 500/401 with nil DB, got %d", w.Code)
	}
}

func TestIntegrationHandlerHealthCheckNilDB(t *testing.T) {
	r := integrationRouter()
	req, _ := http.NewRequest("POST", "/api/v1/integrations/connectors/1/health-check", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError && w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 500/401 with nil DB, got %d", w.Code)
	}
}
