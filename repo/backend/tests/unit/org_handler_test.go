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

func orgRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	h := handlers.NewOrgHandler(nil)
	g := r.Group("/api/v1")
	g.POST("/org/nodes", h.CreateNode)
	g.GET("/org/nodes/:id", h.GetNode)
	g.PUT("/org/nodes/:id", h.UpdateNode)
	g.DELETE("/org/nodes/:id", h.DeleteNode)
	return r
}

func TestOrgHandlerCreateNodeInvalidBody(t *testing.T) {
	r := orgRouter()
	req, _ := http.NewRequest("POST", "/api/v1/org/nodes", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["message"] == nil {
		t.Error("expected validation error message")
	}
}

func TestOrgHandlerGetNodeInvalidID(t *testing.T) {
	r := orgRouter()
	req, _ := http.NewRequest("GET", "/api/v1/org/nodes/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestOrgHandlerUpdateNodeInvalidID(t *testing.T) {
	r := orgRouter()
	b, _ := json.Marshal(map[string]string{"name": "X"})
	req, _ := http.NewRequest("PUT", "/api/v1/org/nodes/xyz", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestOrgHandlerDeleteNodeInvalidID(t *testing.T) {
	r := orgRouter()
	req, _ := http.NewRequest("DELETE", "/api/v1/org/nodes/bad", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
