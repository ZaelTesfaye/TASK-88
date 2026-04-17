package unit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	appErrors "backend/internal/errors"
	"backend/internal/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------- RequestIDMiddleware ----------

func TestRequestIDMiddlewareGeneratesID(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.GET("/test", func(c *gin.Context) {
		cid, _ := c.Get("correlation_id")
		c.JSON(200, gin.H{"cid": cid})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	cid := w.Header().Get("X-Correlation-ID")
	if cid == "" {
		t.Error("expected X-Correlation-ID header to be set")
	}
}

func TestRequestIDMiddlewarePassesThrough(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.GET("/test", func(c *gin.Context) {
		cid, _ := c.Get("correlation_id")
		c.JSON(200, gin.H{"cid": cid})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Correlation-ID", "my-custom-id-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	cid := w.Header().Get("X-Correlation-ID")
	if cid != "my-custom-id-123" {
		t.Errorf("expected 'my-custom-id-123', got %q", cid)
	}
}

func TestRequestIDMiddlewareSetsContext(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())

	var gotCID string
	r.GET("/test", func(c *gin.Context) {
		val, _ := c.Get("correlation_id")
		gotCID, _ = val.(string)
		c.Status(200)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Correlation-ID", "ctx-test-id")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if gotCID != "ctx-test-id" {
		t.Errorf("expected context correlation_id 'ctx-test-id', got %q", gotCID)
	}
}

// ---------- RecoveryMiddleware ----------

func TestRecoveryMiddlewareCatchesPanic(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.RecoveryMiddleware())
	r.GET("/panic", func(c *gin.Context) {
		panic("test panic!")
	})

	req, _ := http.NewRequest("GET", "/panic", nil)
	req.Header.Set("X-Correlation-ID", "panic-test-id")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	// Verify response body has AppError structure.
	var body map[string]interface{}
	if err := parseJSON(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["code"] != "INTERNAL_ERROR" {
		t.Errorf("expected code 'INTERNAL_ERROR', got %v", body["code"])
	}
	if body["message"] == nil || body["message"] == "" {
		t.Error("expected non-empty error message")
	}
	if body["correlationId"] != "panic-test-id" {
		t.Errorf("expected correlationId 'panic-test-id', got %v", body["correlationId"])
	}
}

func TestRecoveryMiddlewareNoPanic(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RecoveryMiddleware())
	r.GET("/ok", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/ok", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ---------- EgressGuardMiddleware ----------

func TestEgressGuardAllowsLocalhost(t *testing.T) {
	r := gin.New()
	r.Use(appErrors.ErrorHandlerMiddleware())
	r.Use(middleware.EgressGuardMiddleware([]string{"localhost", "127.0.0.1"}))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200 for localhost, got %d", w.Code)
	}
}

func TestEgressGuardAllowsCIDR(t *testing.T) {
	r := gin.New()
	r.Use(appErrors.ErrorHandlerMiddleware())
	r.Use(middleware.EgressGuardMiddleware([]string{"192.168.0.0/16"}))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200 for IP in CIDR, got %d", w.Code)
	}
}

func TestEgressGuardBlocksUnknownIP(t *testing.T) {
	r := gin.New()
	r.Use(appErrors.ErrorHandlerMiddleware())
	r.Use(middleware.EgressGuardMiddleware([]string{"10.0.0.0/8"}))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "203.0.113.50:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for blocked IP, got %d", w.Code)
	}
}

// ---------- helper ----------

func parseJSON(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}
