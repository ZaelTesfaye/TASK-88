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

func playbackRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	h := handlers.NewPlaybackHandler(nil)
	g := r.Group("/api/v1/media")
	g.GET("/:id", h.GetMedia)
	g.GET("/:id/stream", h.StreamAudio)
	g.GET("/:id/cover", h.GetCoverArt)
	g.GET("/:id/lyrics/search", h.SearchLyrics)
	g.POST("/:id/lyrics/parse", h.ParseLyrics)
	g.PUT("/:id", h.UpdateMedia)
	g.DELETE("/:id", h.DeleteMedia)
	return r
}

func TestPlaybackHandlerGetMediaInvalidID(t *testing.T) {
	r := playbackRouter()
	req, _ := http.NewRequest("GET", "/api/v1/media/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["message"] == nil {
		t.Error("expected error message for invalid media ID")
	}
}

func TestPlaybackHandlerStreamInvalidID(t *testing.T) {
	r := playbackRouter()
	req, _ := http.NewRequest("GET", "/api/v1/media/xyz/stream", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPlaybackHandlerCoverInvalidID(t *testing.T) {
	r := playbackRouter()
	req, _ := http.NewRequest("GET", "/api/v1/media/bad/cover", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPlaybackHandlerUpdateInvalidID(t *testing.T) {
	r := playbackRouter()
	b, _ := json.Marshal(map[string]string{"title": "X"})
	req, _ := http.NewRequest("PUT", "/api/v1/media/nope", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPlaybackHandlerDeleteInvalidID(t *testing.T) {
	r := playbackRouter()
	req, _ := http.NewRequest("DELETE", "/api/v1/media/oops", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPlaybackHandlerLyricsSearchInvalidID(t *testing.T) {
	r := playbackRouter()
	req, _ := http.NewRequest("GET", "/api/v1/media/nah/lyrics/search?q=hello", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPlaybackHandlerLyricsParseInvalidID(t *testing.T) {
	r := playbackRouter()
	b, _ := json.Marshal(map[string]string{"content": "[00:00.00]Test"})
	req, _ := http.NewRequest("POST", "/api/v1/media/bad/lyrics/parse", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
