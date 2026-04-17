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

// These tests exercise the real production VersionHandler code.
// They require TEST_DB_DSN for full handler paths that hit the DB.

func versionRouter(db interface{}) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())

	// The handler will use db=nil for input-validation tests.
	h := handlers.NewVersionHandler(nil)
	g := r.Group("/api/v1/versions")
	{
		g.GET("/:entity/:id", h.GetVersion)
		g.POST("/:entity", h.CreateVersion)
		g.POST("/:entity/:id/items", h.AddVersionItem)
		g.DELETE("/:entity/:id/items/:itemId", h.RemoveVersionItem)
		g.GET("/:entity/:id/diff", h.DiffVersions)
	}
	return r
}

func TestVersionHandlerGetVersionInvalidID(t *testing.T) {
	r := versionRouter(nil)
	req, _ := http.NewRequest("GET", "/api/v1/versions/sku/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["message"] == nil {
		t.Error("expected error message for invalid ID")
	}
}

func TestVersionHandlerGetVersionNegativeID(t *testing.T) {
	r := versionRouter(nil)
	req, _ := http.NewRequest("GET", "/api/v1/versions/sku/-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestVersionHandlerCreateVersionMissingScopeKey(t *testing.T) {
	r := versionRouter(nil)
	body, _ := json.Marshal(map[string]interface{}{"wrong_field": "x"})
	req, _ := http.NewRequest("POST", "/api/v1/versions/sku", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["message"] == nil {
		t.Error("expected error message about missing scope_key")
	}
}

func TestVersionHandlerCreateVersionEmptyBody(t *testing.T) {
	r := versionRouter(nil)
	req, _ := http.NewRequest("POST", "/api/v1/versions/sku", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestVersionHandlerAddItemsMissingRecordIDs(t *testing.T) {
	r := versionRouter(nil)
	body, _ := json.Marshal(map[string]interface{}{})
	req, _ := http.NewRequest("POST", "/api/v1/versions/sku/1/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestVersionHandlerRemoveItemInvalidItemID(t *testing.T) {
	r := versionRouter(nil)
	req, _ := http.NewRequest("DELETE", "/api/v1/versions/sku/1/items/notanumber", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestVersionHandlerDiffMissingCompareTo(t *testing.T) {
	r := versionRouter(nil)
	req, _ := http.NewRequest("GET", "/api/v1/versions/sku/1/diff", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["message"] == nil {
		t.Error("expected error message about missing compare_to")
	}
}

func TestVersionHandlerDiffInvalidCompareTo(t *testing.T) {
	r := versionRouter(nil)
	req, _ := http.NewRequest("GET", "/api/v1/versions/sku/1/diff?compare_to=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
